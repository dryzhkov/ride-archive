package httpapi

import (
	"net/http"
	"strconv"
)

func (a *API) parseSource(w http.ResponseWriter, r *http.Request) {
	paths, err := a.store.ParseSource(r.Context(), a.owner, r.PathValue("id"))
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, map[string]any{"items": paths})
}
func (a *API) sourcePaths(w http.ResponseWriter, r *http.Request) {
	paths, err := a.store.SourcePaths(r.Context(), a.owner, r.PathValue("id"))
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, map[string]any{"items": paths})
}
func (a *API) entryPaths(w http.ResponseWriter, r *http.Request) {
	paths, err := a.store.EntryPaths(r.Context(), a.owner, r.PathValue("id"))
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, map[string]any{"items": paths})
}
func (a *API) entryMap(w http.ResponseWriter, r *http.Request) {
	paths, err := a.store.EntryMapPaths(r.Context(), a.owner, r.PathValue("id"))
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, map[string]any{"items": paths})
}
func (a *API) attachPath(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PathID   string `json:"path_id"`
		Revision int    `json:"revision"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.PathID == "" || body.Revision < 1 {
		fail(w, 400, "invalid_link", "Provide path_id and current entry revision.")
		return
	}
	e, err := a.store.SetPathLink(r.Context(), a.owner, r.PathValue("id"), body.PathID, body.Revision, true)
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, e)
}
func (a *API) detachPath(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	revision, err := strconv.Atoi(q.Get("revision"))
	if err != nil || revision < 1 || len(q) != 1 || len(q["revision"]) != 1 {
		fail(w, 400, "invalid_link", "Provide the current entry revision.")
		return
	}
	e, err := a.store.SetPathLink(r.Context(), a.owner, r.PathValue("id"), r.PathValue("pathID"), revision, false)
	if err != nil {
		problem(w, err)
		return
	}
	reply(w, 200, e)
}
