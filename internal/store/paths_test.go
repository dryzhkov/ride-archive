package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	"ridearchive/internal/gpx"
)

const multiGPX = `<gpx><trk><name>Day one</name><trkseg><trkpt lat="45" lon="-120"/></trkseg><trkseg><trkpt lat="46" lon="-121"/></trkseg></trk><trk><name>Day two</name><trkseg><trkpt lat="47" lon="-122"/></trkseg></trk></gpx>`

func TestParseAndLinkLifecycle(t *testing.T) {
	s, ctx, path := fixture(t)
	source, _, err := s.PreserveSource(ctx, "alice", "multi.gpx", []byte(multiGPX))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ParseSource(ctx, "bob", source.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	paths, err := s.ParseSource(ctx, "alice", source.ID)
	if err != nil || len(paths) != 2 {
		t.Fatal(paths, err)
	}
	repeated, err := s.ParseSource(ctx, "alice", source.ID)
	if err != nil || paths[0].ID != repeated[0].ID {
		t.Fatal(repeated, err)
	}
	var count int
	if err = s.db.QueryRowContext(ctx, "SELECT count(*) FROM source_parses").Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	var geometry string
	if err = s.db.QueryRowContext(ctx, "SELECT geometry_json FROM paths WHERE id=?", paths[0].ID).Scan(&geometry); err != nil || paths[0].SegmentCount != 2 {
		t.Fatal(err, geometry)
	}
	e, err := s.CreateEntry(ctx, "alice", "First day", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	e, err = s.SetPathLink(ctx, "alice", e.ID, paths[0].ID, e.Revision, true)
	if err != nil || e.Revision != 2 {
		t.Fatal(e, err)
	}
	duplicate, err := s.SetPathLink(ctx, "alice", e.ID, paths[0].ID, e.Revision, true)
	if err != nil || duplicate.Revision != 2 {
		t.Fatal(duplicate, err)
	}
	if _, err = s.SetPathLink(ctx, "alice", e.ID, paths[1].ID, 1, true); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	links, err := s.EntryPaths(ctx, "alice", e.ID)
	if err != nil || len(links) != 1 || links[0].Name != "Day one" {
		t.Fatal(links, err)
	}
	other, err := s.CreateEntry(ctx, "alice", "Another entry", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SetPathLink(ctx, "alice", other.ID, paths[0].ID, other.Revision, true); err != nil {
		t.Fatal(err)
	}
	foreign, err := s.CreateEntry(ctx, "bob", "Other owner", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SetPathLink(ctx, "bob", foreign.ID, paths[0].ID, foreign.Revision, true); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, "INSERT INTO entry_paths VALUES (?,?,?,?)", "bob", foreign.ID, paths[0].ID, now()); err == nil {
		t.Fatal("cross-owner database link allowed")
	}
	e, err = s.SetPathLink(ctx, "alice", e.ID, paths[0].ID, e.Revision, false)
	if err != nil || e.Revision != 3 {
		t.Fatal(e, err)
	}
	raw, err := s.SourceBytes(ctx, "alice", source.ID)
	if err != nil || string(raw) != multiGPX {
		t.Fatal("source modified", err)
	}
	if err = s.db.QueryRowContext(ctx, "SELECT count(*) FROM path_link_changes WHERE entry_id=?", e.ID).Scan(&count); err != nil || count != 2 {
		t.Fatal(count, err)
	}
	s.Close()
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	links, err = reopened.EntryPaths(ctx, "alice", other.ID)
	if err != nil || len(links) != 1 {
		t.Fatal(links, err)
	}
}
func TestInvalidParseLeavesOriginalAndNoPartialParse(t *testing.T) {
	s, ctx, _ := fixture(t)
	raw := []byte(`<gpx><trk><trkseg><trkpt lat="1" lon="2"/></trkseg></trk><rte><rtept lat="bad" lon="2"/></rte></gpx>`)
	source, _, err := s.PreserveSource(ctx, "alice", "bad.gpx", raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ParseSource(ctx, "alice", source.ID); !errors.Is(err, gpx.ErrInvalid) {
		t.Fatal(err)
	}
	for _, table := range []string{"paths", "source_parses"} {
		var count int
		if err = s.db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatal(table, count, err)
		}
	}
	kept, err := s.SourceBytes(ctx, "alice", source.ID)
	if err != nil || string(kept) != string(raw) {
		t.Fatal(err)
	}
}
func TestUpgradeExistingV1Database(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := migrations.ReadFile("migrations/001_core.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	// Let migrate record the old migration with its real checksum.
	if _, err = db.Exec("CREATE TABLE schema_migrations(name TEXT PRIMARY KEY,sha256 TEXT NOT NULL) STRICT"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO schema_migrations VALUES ('001_core.sql',?)", checksumForTest(schema)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO users VALUES ('alice','Alice'); INSERT INTO trips VALUES ('trip','alice','Existing trip','2026-01-01T00:00:00Z'); INSERT INTO archive_entries VALUES ('entry','alice','Existing entry','unclassified','trip',NULL,1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	e, err := s.Entry(ctx, "alice", "entry")
	if err != nil || e.TripID == nil || *e.TripID != "trip" || e.Revision != 1 {
		t.Fatal(e, err)
	}
	if err = s.migrate(ctx); err != nil {
		t.Fatal(err)
	}
}

func checksumForTest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
