// Package httpapi maps versioned JSON contracts onto the archive repository.
package httpapi

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"ridearchive/internal/domain"
	"ridearchive/internal/gpx"
	"ridearchive/internal/store"
)

type API struct {
	store        *store.Store
	owner, token string
}

func New(s *store.Store, owner, token string, static http.Handler) http.Handler {
	a := &API{store: s, owner: owner, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { reply(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Ping(r.Context()); err != nil {
			fail(w, 503, "not_ready", "Database is unavailable.")
			return
		}
		reply(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/me", a.me)
	for _, kind := range []string{"trips", "bikes"} {
		mux.HandleFunc("GET /api/v1/"+kind, a.listCatalog(kind))
		mux.HandleFunc("POST /api/v1/"+kind, a.createCatalog(kind))
	}
	mux.HandleFunc("GET /api/v1/entries", a.listEntries)
	mux.HandleFunc("POST /api/v1/entries", a.createEntry)
	mux.HandleFunc("GET /api/v1/entries/{id}", a.getEntry)
	mux.HandleFunc("PATCH /api/v1/entries/{id}", a.patchEntry)
	mux.HandleFunc("GET /api/v1/entries/{id}/history", a.history)
	mux.HandleFunc("GET /api/v1/sources", a.listSources)
	mux.HandleFunc("POST /api/v1/sources", a.uploadSource)
	mux.HandleFunc("GET /api/v1/sources/{id}/original", a.original)
	mux.HandleFunc("POST /api/v1/sources/{id}/parse", a.parseSource)
	mux.HandleFunc("GET /api/v1/sources/{id}/paths", a.sourcePaths)
	mux.HandleFunc("GET /api/v1/entries/{id}/paths", a.entryPaths)
	mux.HandleFunc("GET /api/v1/entries/{id}/map", a.entryMap)
	mux.HandleFunc("POST /api/v1/entries/{id}/sources", a.addEntrySource)
	mux.HandleFunc("POST /api/v1/entries/{id}/paths", a.attachPath)
	mux.HandleFunc("DELETE /api/v1/entries/{id}/paths/{pathID}", a.detachPath)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "not_found", "Unknown API route.") })
	if static != nil {
		mux.Handle("/", static)
	}
	return a.guard(mux)
}
func (a *API) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// The interactive OSM basemap needs an origin Referer to comply with its tile policy.
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		// Local mode also checks Host, preventing a foreign hostname from reaching an
		// unauthenticated loopback API through DNS rebinding.
		if a.token == "" {
			host := r.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			ip := net.ParseIP(host)
			if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
				fail(w, 403, "forbidden", "Local mode requires a loopback host.")
				return
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host != r.Host || u.User != nil {
					fail(w, 403, "forbidden", "Cross-origin writes are not allowed.")
					return
				}
			}
		}
		if a.token != "" && strings.HasPrefix(r.URL.Path, "/api/") {
			expected := sha256.Sum256([]byte("Bearer " + a.token))
			actual := sha256.Sum256([]byte(r.Header.Get("Authorization")))
			if subtle.ConstantTimeCompare(expected[:], actual[:]) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				fail(w, 401, "unauthorized", "An API token is required.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code, message string) {
	reply(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func problem(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		fail(w, 404, "not_found", "Record not found.")
	case errors.Is(err, store.ErrConflict):
		fail(w, 409, "revision_conflict", "Entry changed; reload before editing.")
	case errors.Is(err, gpx.ErrInvalid):
		fail(w, 422, "invalid_gpx", err.Error())
	case errors.Is(err, store.ErrInvalid):
		fail(w, 400, "invalid_record", "Check the name and trip/bike associations.")
	default:
		slog.Error("repository operation failed")
		fail(w, 500, "internal_error", "The operation could not be completed.")
	}
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		fail(w, 415, "unsupported_media_type", "Use application/json.")
		return false
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32768))
	if err != nil {
		fail(w, 413, "body_too_large", "JSON body exceeds 32 KiB.")
		return false
	}
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		fail(w, 400, "invalid_json", "Expected a JSON object.")
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err = d.Decode(v); err != nil {
		fail(w, 400, "invalid_json", "Invalid JSON or unknown fields.")
		return false
	}
	if err = d.Decode(new(any)); err != io.EOF {
		fail(w, 400, "invalid_json", "Expected one JSON object.")
		return false
	}
	return true
}
func pagination(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	q := r.URL.Query()
	for k, v := range q {
		if (k != "limit" && k != "offset") || len(v) != 1 {
			fail(w, 400, "invalid_pagination", "Use limit and offset only.")
			return 0, 0, false
		}
	}
	limit, offset := 50, 0
	var err error
	if q.Has("limit") {
		limit, err = strconv.Atoi(q.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			fail(w, 400, "invalid_pagination", "Limit must be 1–100.")
			return 0, 0, false
		}
	}
	if q.Has("offset") {
		offset, err = strconv.Atoi(q.Get("offset"))
		if err != nil || offset < 0 || offset > 100000 {
			fail(w, 400, "invalid_pagination", "Offset must be 0–100000.")
			return 0, 0, false
		}
	}
	return limit, offset, true
}
func page[T any](w http.ResponseWriter, items []T, limit, offset int) {
	var next *int
	if len(items) > limit {
		items = items[:limit]
		n := offset + limit
		if n <= 100000 {
			next = &n
		}
	}
	reply(w, 200, map[string]any{"items": items, "next_offset": next})
}
func (a *API) me(w http.ResponseWriter, r *http.Request) {
	u, err := a.store.User(r.Context(), a.owner)
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, u)
}
func (a *API) listCatalog(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l, o, ok := pagination(w, r)
		if !ok {
			return
		}
		items, err := a.store.ListCatalog(r.Context(), a.owner, kind, l+1, o)
		if err != nil {
			problem(w, err)
			return
		}
		page(w, items, l, o)
	}
}
func (a *API) createCatalog(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			Name string `json:"name"`
		}
		if !decode(w, r, &p) {
			return
		}
		v, err := a.store.CreateCatalog(r.Context(), a.owner, kind, p.Name)
		if err != nil {
			problem(w, err)
			return
		}
		reply(w, 201, v)
	}
}
func (a *API) listEntries(w http.ResponseWriter, r *http.Request) {
	l, o, ok := pagination(w, r)
	if !ok {
		return
	}
	items, err := a.store.ListEntries(r.Context(), a.owner, l+1, o)
	if err != nil {
		problem(w, err)
		return
	}
	page(w, items, l, o)
}
func (a *API) createEntry(w http.ResponseWriter, r *http.Request) {
	media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if media == "multipart/form-data" {
		p, raw, filename, ok := readGPXForm(w, r)
		if !ok {
			return
		}
		entry, source, paths, err := a.store.SaveEntryWithGPX(r.Context(), a.owner, "", p.Title, p.TripID, p.BikeID, 0, filename, raw)
		if err != nil {
			problem(w, err)
			return
		}
		w.Header().Set("Location", "/api/v1/entries/"+entry.ID)
		reply(w, 201, map[string]any{"entry": entry, "source": source, "paths": paths})
		return
	}
	var p struct {
		Title  string  `json:"title"`
		TripID *string `json:"trip_id"`
		BikeID *string `json:"bike_id"`
	}
	if !decode(w, r, &p) {
		return
	}
	v, err := a.store.CreateEntry(r.Context(), a.owner, p.Title, p.TripID, p.BikeID)
	if err != nil {
		problem(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/entries/"+v.ID)
	reply(w, 201, v)
}
func readGPXForm(w http.ResponseWriter, r *http.Request) (struct {
	Title          string
	TripID, BikeID *string
}, []byte, string, bool) {
	var p struct {
		Title          string
		TripID, BikeID *string
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024+64*1024)
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		fail(w, 413, "request_too_large", "Choose a GPX file no larger than 10 MiB.")
		return p, nil, "", false
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
		fail(w, 400, "invalid_upload", "Choose one GPX file.")
		return p, nil, "", false
	}
	fields := r.MultipartForm.Value
	if len(fields["title"]) > 1 || len(fields["trip_id"]) > 1 || len(fields["bike_id"]) > 1 || len(fields["revision"]) > 1 {
		fail(w, 400, "invalid_upload", "Check the entry details and try again.")
		return p, nil, "", false
	}
	allowed := map[string]bool{"title": true, "trip_id": true, "bike_id": true, "revision": true}
	for key := range fields {
		if !allowed[key] {
			fail(w, 400, "invalid_upload", "Check the entry details and try again.")
			return p, nil, "", false
		}
	}
	if values := fields["title"]; len(values) == 1 {
		p.Title = values[0]
	}
	if values := fields["trip_id"]; len(values) == 1 && values[0] != "" {
		p.TripID = &values[0]
	}
	if values := fields["bike_id"]; len(values) == 1 && values[0] != "" {
		p.BikeID = &values[0]
	}
	f, err := r.MultipartForm.File["file"][0].Open()
	if err != nil {
		fail(w, 400, "invalid_upload", "The GPX file could not be read.")
		return p, nil, "", false
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 10*1024*1024+1))
	if err != nil || len(raw) > 10*1024*1024 {
		fail(w, 413, "request_too_large", "Choose a GPX file no larger than 10 MiB.")
		return p, nil, "", false
	}
	filename := filepath.Base(strings.ReplaceAll(r.MultipartForm.File["file"][0].Filename, "\\", "/"))
	return p, raw, filename, true
}
func (a *API) addEntrySource(w http.ResponseWriter, r *http.Request) {
	p, raw, filename, ok := readGPXForm(w, r)
	if !ok {
		return
	}
	if p.Title != "" || p.TripID != nil || p.BikeID != nil {
		fail(w, 400, "invalid_upload", "Only a GPX file is needed for an existing ride.")
		return
	}
	revision, err := strconv.Atoi(r.FormValue("revision"))
	if err != nil || revision < 1 {
		fail(w, 400, "invalid_revision", "Reload the entry and try again.")
		return
	}
	entry, source, paths, err := a.store.SaveEntryWithGPX(r.Context(), a.owner, r.PathValue("id"), "", nil, nil, revision, filename, raw)
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, map[string]any{"entry": entry, "source": source, "paths": paths})
}
func (a *API) getEntry(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.Entry(r.Context(), a.owner, r.PathValue("id"))
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, v)
}
func (a *API) patchEntry(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Revision int             `json:"revision"`
		Title    json.RawMessage `json:"title"`
		TripID   json.RawMessage `json:"trip_id"`
		BikeID   json.RawMessage `json:"bike_id"`
	}
	if !decode(w, r, &p) {
		return
	}
	if p.Revision < 1 || (p.Title == nil && p.TripID == nil && p.BikeID == nil) {
		fail(w, 400, "invalid_patch", "Provide revision and at least one editable field.")
		return
	}
	patch := domain.EntryPatch{Revision: p.Revision, TripSet: p.TripID != nil, BikeSet: p.BikeID != nil}
	if p.Title != nil {
		var title string
		if string(p.Title) == "null" || json.Unmarshal(p.Title, &title) != nil {
			fail(w, 400, "invalid_patch", "Title must be a string.")
			return
		}
		patch.Title = &title
	}
	for _, field := range []struct {
		raw   json.RawMessage
		value **string
	}{{p.TripID, &patch.TripID}, {p.BikeID, &patch.BikeID}} {
		if field.raw != nil && json.Unmarshal(field.raw, field.value) != nil {
			fail(w, 400, "invalid_patch", "Associations must be a string or null.")
			return
		}
	}
	v, err := a.store.UpdateEntry(r.Context(), a.owner, r.PathValue("id"), patch)
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, v)
}
func (a *API) history(w http.ResponseWriter, r *http.Request) {
	l, o, ok := pagination(w, r)
	if !ok {
		return
	}
	v, err := a.store.EntryHistory(r.Context(), a.owner, r.PathValue("id"), l+1, o)
	if err != nil {
		problem(w, err)
		return
	}
	page(w, v, l, o)
}
func (a *API) listSources(w http.ResponseWriter, r *http.Request) {
	l, o, ok := pagination(w, r)
	if !ok {
		return
	}
	v, err := a.store.ListSources(r.Context(), a.owner, l+1, o)
	if err != nil {
		problem(w, err)
		return
	}
	page(w, v, l, o)
}
func (a *API) uploadSource(w http.ResponseWriter, r *http.Request) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/gpx+xml" {
		fail(w, 415, "unsupported_media_type", "Use application/gpx+xml with raw original bytes.")
		return
	}
	filename := r.Header.Get("X-Filename")
	if filename == "" || strings.ContainsAny(filename, "/\\\r\n") {
		fail(w, 400, "invalid_filename", "Provide a filename in X-Filename, without a directory.")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10*1024*1024))
	if err != nil {
		fail(w, 413, "body_too_large", "Original file exceeds 10 MiB.")
		return
	}
	v, duplicate, err := a.store.PreserveSource(r.Context(), a.owner, filename, raw)
	if err != nil {
		problem(w, err)
		return
	}
	status := 201
	if duplicate {
		status = 200
	}
	reply(w, status, map[string]any{"source": v, "duplicate": duplicate, "parse_status": "not_started"})
}
func (a *API) original(w http.ResponseWriter, r *http.Request) {
	raw, err := a.store.SourceBytes(r.Context(), a.owner, r.PathValue("id"))
	if err != nil {
		problem(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/gpx+xml")
	w.Header().Set("Content-Disposition", `attachment; filename="original.gpx"`)
	w.WriteHeader(200)
	_, _ = w.Write(raw)
}
