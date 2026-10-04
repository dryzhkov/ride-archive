package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"ridearchive/internal/domain"
	"ridearchive/internal/store"
)

func setup(t *testing.T, token string) http.Handler {
	t.Helper()
	s, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err = s.EnsureUser(context.Background(), "owner", "Test"); err != nil {
		t.Fatal(err)
	}
	return New(s, "owner", token, nil)
}
func send(h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestCoreWorkflow(t *testing.T) {
	h := setup(t, "")
	tripResult := send(h, "POST", "/api/v1/trips", `{"name":"Weekend"}`, nil)
	if tripResult.Code != 201 {
		t.Fatal(tripResult.Code, tripResult.Body)
	}
	var trip domain.CatalogItem
	json.Unmarshal(tripResult.Body.Bytes(), &trip)
	result := send(h, "POST", "/api/v1/entries", `{"title":"First day","trip_id":"`+trip.ID+`"}`, nil)
	if result.Code != 201 {
		t.Fatal(result.Code, result.Body)
	}
	var e domain.ArchiveEntry
	json.Unmarshal(result.Body.Bytes(), &e)
	if e.Kind != domain.Unclassified || e.Revision != 1 {
		t.Fatal(e)
	}
	patch := send(h, "PATCH", "/api/v1/entries/"+e.ID, `{"revision":1,"trip_id":null,"title":"Updated"}`, nil)
	if patch.Code != 200 {
		t.Fatal(patch.Code, patch.Body)
	}
	stale := send(h, "PATCH", "/api/v1/entries/"+e.ID, `{"revision":1,"title":"Stale"}`, nil)
	if stale.Code != 409 {
		t.Fatal(stale.Code)
	}
	history := send(h, "GET", "/api/v1/entries/"+e.ID+"/history", "", nil)
	if history.Code != 200 || !strings.Contains(history.Body.String(), `"before"`) {
		t.Fatal(history.Code, history.Body)
	}
	result = send(h, "GET", "/api/v1/entries", "", nil)
	if result.Code != 200 || !strings.Contains(result.Body.String(), `"trip_id":null`) {
		t.Fatal(result.Code, result.Body)
	}
}
func TestValidation(t *testing.T) {
	h := setup(t, "")
	for _, tc := range []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/api/v1/entries", `{"title":"test","owner_id":"other"}`, 400},
		{"POST", "/api/v1/entries", `{"title":"test","kind":"recorded"}`, 400},
		{"POST", "/api/v1/entries", `{"title":"test","trip_id":"missing"}`, 400},
		{"POST", "/api/v1/entries", `{"title":" "}`, 400},
		{"POST", "/api/v1/entries", `null`, 400},
		{"POST", "/api/v1/entries", `{"title":"test"}{}`, 400},
		{"POST", "/api/v1/entries", `{"title":"` + strings.Repeat("a", 33000) + `"}`, 413},
		{"GET", "/api/v1/entries?limit=101", "", 400},
		{"GET", "/api/v1/entries?offset=-1", "", 400},
		{"GET", "/api/v1/entries?owner_id=other", "", 400},
		{"GET", "/api/v1/entries?limit=2&limit=3", "", 400},
		{"GET", "/api/v1/entries/missing", "", 404},
		{"PATCH", "/api/v1/entries/missing", `{"revision":1,"title":null}`, 400},
		{"PATCH", "/api/v1/entries/missing", `{"revision":1}`, 400},
	} {
		w := send(h, tc.method, tc.path, tc.body, nil)
		if w.Code != tc.want {
			t.Errorf("%s %s: %d want %d: %s", tc.method, tc.path, w.Code, tc.want, w.Body)
		}
	}
}
func TestAuthAndOrigin(t *testing.T) {
	token := strings.Repeat("a", 32)
	h := setup(t, token)
	if w := send(h, "GET", "/api/v1/me", "", nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := send(h, "GET", "/api/v1/me", "", map[string]string{"Authorization": "Bearer " + token}); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := send(h, "POST", "/api/v1/trips", `{"name":"no"}`, map[string]string{"Authorization": "Bearer " + token, "Origin": "https://other.example"}); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := send(h, "GET", "/healthz", "", nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	local := setup(t, "")
	r := httptest.NewRequest("GET", "http://attacker.example/api/v1/me", nil)
	w := httptest.NewRecorder()
	local.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("foreign host accepted")
	}
}
func TestUploadRoundTripAndPagination(t *testing.T) {
	h := setup(t, "")
	raw := "<gpx>\r\n</gpx>"
	headers := map[string]string{"Content-Type": "application/gpx+xml", "X-Filename": "test.gpx"}
	w := send(h, "POST", "/api/v1/sources", raw, headers)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body)
	}
	var created struct {
		Source domain.Source `json:"source"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)
	w = send(h, "GET", "/api/v1/sources/"+created.Source.ID+"/original", "", nil)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), []byte(raw)) {
		t.Fatal(w.Code, w.Body)
	}
	w = send(h, "POST", "/api/v1/sources", raw, headers)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"duplicate":true`) {
		t.Fatal(w.Code, w.Body)
	}
	for range 3 {
		w = send(h, "POST", "/api/v1/entries", `{"title":"Entry"}`, nil)
		if w.Code != 201 {
			t.Fatal(w.Code)
		}
	}
	w = send(h, "GET", "/api/v1/entries?limit=2", "", nil)
	if !strings.Contains(w.Body.String(), `"next_offset":2`) {
		t.Fatal(w.Body)
	}
	w = send(h, "GET", "/api/v1/entries?limit=2&offset=2", "", nil)
	if !strings.Contains(w.Body.String(), `"next_offset":null`) {
		t.Fatal(w.Body)
	}
}
