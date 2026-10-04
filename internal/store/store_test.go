package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	"ridearchive/internal/domain"
)

func fixture(t *testing.T) (*Store, context.Context, string) {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "archive.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	for _, id := range []string{"alice", "bob"} {
		if err = s.EnsureUser(ctx, id, id); err != nil {
			t.Fatal(err)
		}
	}
	return s, ctx, path
}
func TestEntriesOwnershipHistoryAndPersistence(t *testing.T) {
	s, ctx, path := fixture(t)
	trip, err := s.CreateCatalog(ctx, "alice", "trips", "A trip")
	if err != nil {
		t.Fatal(err)
	}
	bike, err := s.CreateCatalog(ctx, "bob", "bikes", "Other bike")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateEntry(ctx, "alice", "Bad link", nil, &bike.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-owner create: %v", err)
	}
	e, err := s.CreateEntry(ctx, "alice", "An outing", &trip.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if e.Kind != domain.Unclassified || e.BikeID != nil {
		t.Fatalf("unexpected evidence: %+v", e)
	}
	if _, err = s.Entry(ctx, "bob", e.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = s.UpdateEntry(ctx, "bob", e.ID, domain.EntryPatch{Revision: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = s.UpdateEntry(ctx, "alice", e.ID, domain.EntryPatch{Revision: 1, BikeSet: true, BikeID: &bike.ID}); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	title := "A better title"
	e, err = s.UpdateEntry(ctx, "alice", e.ID, domain.EntryPatch{Revision: 1, Title: &title, TripSet: true})
	if err != nil {
		t.Fatal(err)
	}
	if e.Revision != 2 || e.TripID != nil {
		t.Fatalf("patch failed: %+v", e)
	}
	if _, err = s.UpdateEntry(ctx, "alice", e.ID, domain.EntryPatch{Revision: 1, Title: &title}); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	history, err := s.EntryHistory(ctx, "alice", e.ID, 10, 0)
	if err != nil || len(history) != 2 {
		t.Fatalf("history %v %v", history, err)
	}
	if history[1].ActorID != "alice" || string(history[0].Before) != "null" {
		t.Fatal(history)
	}
	if _, err = s.db.ExecContext(ctx, "UPDATE archive_entries SET bike_id=? WHERE id=?", bike.ID, e.ID); err == nil {
		t.Fatal("database allowed cross-owner bike")
	}
	if _, err = s.db.ExecContext(ctx, "DELETE FROM entry_changes WHERE entry_id=?", e.ID); err == nil {
		t.Fatal("database allowed history deletion")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.Entry(ctx, "alice", e.ID)
	if err != nil || got.Title != title || got.Revision != 2 {
		t.Fatalf("reopen %+v %v", got, err)
	}
}
func TestSourceBytesDuplicateAndOwnerIsolation(t *testing.T) {
	s, ctx, _ := fixture(t)
	raw := []byte("<?xml version=\"1.0\"?>\r\n<gpx/>\n")
	first, duplicate, err := s.PreserveSource(ctx, "alice", "first.gpx", raw)
	if err != nil || duplicate {
		t.Fatal(err, duplicate)
	}
	sum := sha256.Sum256(raw)
	if first.OriginalFilename != "first.gpx" {
		t.Fatal(first)
	}
	if first.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("hash mismatch")
	}
	second, duplicate, err := s.PreserveSource(ctx, "alice", "renamed.gpx", raw)
	if err != nil || !duplicate || second.ID != first.ID {
		t.Fatal(err, duplicate, second)
	}
	if second.OriginalFilename != "first.gpx" {
		t.Fatal(second)
	}
	var count int
	s.db.QueryRowContext(ctx, "SELECT count(*) FROM source_imports WHERE source_id=?", first.ID).Scan(&count)
	if count != 2 {
		t.Fatal(count)
	}
	if _, err = s.SourceBytes(ctx, "bob", first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	other, _, err := s.PreserveSource(ctx, "bob", "first.gpx", raw)
	if err != nil || other.ID == first.ID {
		t.Fatal(err, other)
	}
	got, err := s.SourceBytes(ctx, "alice", first.ID)
	if err != nil || string(got) != string(raw) {
		t.Fatal("original not preserved", err)
	}
	if _, err = s.db.ExecContext(ctx, "UPDATE sources SET original=? WHERE id=?", []byte("changed"), first.ID); err == nil {
		t.Fatal("source was mutable")
	}
}
func TestMigrationVerification(t *testing.T) {
	s, ctx, path := fixture(t)
	if err := s.migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE schema_migrations SET sha256='changed'"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if reopened, err := Open(ctx, path); err == nil {
		reopened.Close()
		t.Fatal("accepted modified migration")
	}
}
func TestFailedEntryIsAtomicAndListsAreScoped(t *testing.T) {
	s, ctx, _ := fixture(t)
	bad := "missing"
	if _, err := s.CreateEntry(ctx, "alice", "Bad", &bad, nil); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM entry_changes").Scan(&count); err != nil || count != 0 {
		t.Fatal(count, err)
	}
	for _, owner := range []string{"alice", "alice", "bob"} {
		if _, err := s.CreateEntry(ctx, owner, "Entry", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	a, err := s.ListEntries(ctx, "alice", 1, 0)
	if err != nil || len(a) != 1 {
		t.Fatal(err, a)
	}
	b, err := s.ListEntries(ctx, "alice", 1, 1)
	if err != nil || len(b) != 1 || b[0].ID == a[0].ID {
		t.Fatal(err, b)
	}
	empty, err := s.ListEntries(ctx, "alice", 1, 2)
	if err != nil || len(empty) != 0 {
		t.Fatal(err, empty)
	}
}
