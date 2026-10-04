package httpapi

import (
	"encoding/json"
	"testing"

	"ridearchive/internal/domain"
)

func TestUploadParseAttachAndDetachAPI(t *testing.T) {
	h := setup(t, "")
	headers := map[string]string{"Content-Type": "application/gpx+xml", "X-Filename": "outing.gpx"}
	w := send(h, "POST", "/api/v1/sources", `<gpx><trk><name>A track</name><trkseg><trkpt lat="1" lon="2"/></trkseg></trk></gpx>`, headers)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body)
	}
	var uploaded struct {
		Source domain.Source `json:"source"`
	}
	json.Unmarshal(w.Body.Bytes(), &uploaded)
	w = send(h, "POST", "/api/v1/sources/"+uploaded.Source.ID+"/parse", "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	var paths struct {
		Items []domain.PathSummary `json:"items"`
	}
	json.Unmarshal(w.Body.Bytes(), &paths)
	if len(paths.Items) != 1 {
		t.Fatal(paths)
	}
	w = send(h, "POST", "/api/v1/entries", `{"title":"Outing"}`, nil)
	var e domain.ArchiveEntry
	json.Unmarshal(w.Body.Bytes(), &e)
	url := "/api/v1/entries/" + e.ID + "/paths"
	w = send(h, "POST", url, `{"path_id":"`+paths.Items[0].ID+`","revision":1}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = send(h, "GET", url, "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var links struct {
		Items []domain.PathSummary `json:"items"`
	}
	json.Unmarshal(w.Body.Bytes(), &links)
	if len(links.Items) != 1 {
		t.Fatal(links)
	}
	w = send(h, "DELETE", url+"/"+paths.Items[0].ID+"?revision=1", "", nil)
	if w.Code != 409 {
		t.Fatal(w.Code)
	}
	w = send(h, "DELETE", url+"/"+paths.Items[0].ID+"?revision=2", "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = send(h, "GET", url, "", nil)
	json.Unmarshal(w.Body.Bytes(), &links)
	if len(links.Items) != 0 {
		t.Fatal(links)
	}
	w = send(h, "POST", "/api/v1/sources", `<html/>`, headers)
	json.Unmarshal(w.Body.Bytes(), &uploaded)
	w = send(h, "POST", "/api/v1/sources/"+uploaded.Source.ID+"/parse", "", nil)
	if w.Code != 422 {
		t.Fatal(w.Code, w.Body)
	}
}
