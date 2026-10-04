// Package store owns SQLite queries and embedded migrations.
package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"ridearchive/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS
var (
	ErrNotFound = errors.New("record not found")
	ErrConflict = errors.New("revision conflict")
	ErrInvalid  = errors.New("invalid record")
)

type Store struct{ db *sql.DB }

func Open(ctx context.Context, path string) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: absolute}
	q := u.Query()
	for _, p := range []string{"foreign_keys(1)", "busy_timeout(5000)", "journal_mode(WAL)", "synchronous(FULL)"} {
		q.Add("_pragma", p)
	}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	// One connection serializes writers for the initial personal archive.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{db: db}
	if err = db.PingContext(ctx); err == nil {
		err = s.migrate(ctx)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) migrate(ctx context.Context) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	var known, count int
	if err = conn.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&known); err != nil {
		return err
	}
	if known == 0 {
		if err = conn.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("database is not an empty Ride Archive database; use a new database path")
		}
	}
	if _, err = conn.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, sha256 TEXT NOT NULL) STRICT"); err != nil {
		return err
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	if err = conn.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil {
		return err
	}
	if count > len(files) {
		return errors.New("database schema is newer than this application")
	}
	for _, f := range files {
		raw, err := migrations.ReadFile("migrations/" + f.Name())
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		checksum := hex.EncodeToString(sum[:])
		var previous string
		err = conn.QueryRowContext(ctx, "SELECT sha256 FROM schema_migrations WHERE name=?", f.Name()).Scan(&previous)
		if err == nil {
			if previous != checksum {
				return fmt.Errorf("migration checksum mismatch: %s", f.Name())
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err = conn.ExecContext(ctx, string(raw)); err != nil {
			return fmt.Errorf("migration %s: %w", f.Name(), err)
		}
		if _, err = conn.ExecContext(ctx, "INSERT INTO schema_migrations VALUES (?,?)", f.Name(), checksum); err != nil {
			return err
		}
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}
func now() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z") }
func validName(v string) bool {
	return strings.TrimSpace(v) != "" && utf8.RuneCountInString(v) <= 200 && !strings.ContainsRune(v, '\x00')
}
func (s *Store) EnsureUser(ctx context.Context, id, name string) error {
	if id == "" || !validName(name) {
		return ErrInvalid
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO users(id,display_name) VALUES (?,?) ON CONFLICT(id) DO NOTHING", id, name)
	return err
}
func (s *Store) User(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, "SELECT id,display_name FROM users WHERE id=?", id).Scan(&u.ID, &u.DisplayName)
	return u, notFound(err)
}
func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
func catalogTable(kind string) (string, error) {
	switch kind {
	case "trips", "bikes":
		return kind, nil
	}
	return "", ErrInvalid
}
func (s *Store) CreateCatalog(ctx context.Context, owner, kind, name string) (domain.CatalogItem, error) {
	item := domain.CatalogItem{ID: uuid.NewString(), OwnerID: owner, Name: strings.TrimSpace(name), CreatedAt: now()}
	table, err := catalogTable(kind)
	if err != nil || !validName(name) {
		return item, ErrInvalid
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO "+table+"(id,owner_id,name,created_at) VALUES (?,?,?,?)", item.ID, owner, item.Name, item.CreatedAt)
	return item, err
}
func (s *Store) ListCatalog(ctx context.Context, owner, kind string, limit, offset int) ([]domain.CatalogItem, error) {
	items := []domain.CatalogItem{}
	table, err := catalogTable(kind)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT id,owner_id,name,created_at FROM "+table+" WHERE owner_id=? ORDER BY created_at,id LIMIT ? OFFSET ?", owner, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domain.CatalogItem
		if err = rows.Scan(&v.ID, &v.OwnerID, &v.Name, &v.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

const entryColumns = "id,owner_id,title,kind,trip_id,bike_id,revision,created_at,updated_at"

type scanner interface{ Scan(...any) error }

func scanEntry(row scanner) (domain.ArchiveEntry, error) {
	var v domain.ArchiveEntry
	err := row.Scan(&v.ID, &v.OwnerID, &v.Title, &v.Kind, &v.TripID, &v.BikeID, &v.Revision, &v.CreatedAt, &v.UpdatedAt)
	return v, notFound(err)
}
func validateLinks(ctx context.Context, tx *sql.Tx, owner string, trip, bike *string) error {
	for _, v := range []struct {
		table string
		id    *string
	}{{"trips", trip}, {"bikes", bike}} {
		if v.id == nil {
			continue
		}
		var found int
		err := tx.QueryRowContext(ctx, "SELECT 1 FROM "+v.table+" WHERE owner_id=? AND id=?", owner, *v.id).Scan(&found)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalid
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func recordChange(ctx context.Context, tx *sql.Tx, before *domain.ArchiveEntry, after domain.ArchiveEntry) error {
	var old any
	if before != nil {
		b, err := json.Marshal(before)
		if err != nil {
			return err
		}
		old = string(b)
	}
	b, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO entry_changes(owner_id,entry_id,revision,origin,actor_id,before_json,after_json,recorded_at) VALUES (?,?,?,'user',?,?,?,?)", after.OwnerID, after.ID, after.Revision, after.OwnerID, old, string(b), after.UpdatedAt)
	return err
}
func (s *Store) CreateEntry(ctx context.Context, owner, title string, trip, bike *string) (domain.ArchiveEntry, error) {
	stamp := now()
	v := domain.ArchiveEntry{ID: uuid.NewString(), OwnerID: owner, Title: strings.TrimSpace(title), Kind: domain.Unclassified, TripID: trip, BikeID: bike, Revision: 1, CreatedAt: stamp, UpdatedAt: stamp}
	if !validName(title) {
		return v, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return v, err
	}
	defer tx.Rollback()
	if err = validateLinks(ctx, tx, owner, trip, bike); err != nil {
		return v, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO archive_entries ("+entryColumns+") VALUES (?,?,?,?,?,?,?,?,?)", v.ID, owner, v.Title, v.Kind, trip, bike, 1, stamp, stamp)
	if err != nil {
		return v, err
	}
	if err = recordChange(ctx, tx, nil, v); err != nil {
		return v, err
	}
	return v, tx.Commit()
}
func (s *Store) Entry(ctx context.Context, owner, id string) (domain.ArchiveEntry, error) {
	return scanEntry(s.db.QueryRowContext(ctx, "SELECT "+entryColumns+" FROM archive_entries WHERE owner_id=? AND id=?", owner, id))
}
func (s *Store) ListEntries(ctx context.Context, owner string, limit, offset int) ([]domain.ArchiveEntry, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+entryColumns+" FROM archive_entries WHERE owner_id=? ORDER BY created_at,id LIMIT ? OFFSET ?", owner, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ArchiveEntry{}
	for rows.Next() {
		v, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func (s *Store) UpdateEntry(ctx context.Context, owner, id string, p domain.EntryPatch) (domain.ArchiveEntry, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ArchiveEntry{}, err
	}
	defer tx.Rollback()
	v, err := scanEntry(tx.QueryRowContext(ctx, "SELECT "+entryColumns+" FROM archive_entries WHERE owner_id=? AND id=?", owner, id))
	if err != nil {
		return v, err
	}
	if p.Revision != v.Revision {
		return v, ErrConflict
	}
	before := v
	if p.Title != nil {
		if !validName(*p.Title) {
			return v, ErrInvalid
		}
		v.Title = strings.TrimSpace(*p.Title)
	}
	if p.TripSet {
		v.TripID = p.TripID
	}
	if p.BikeSet {
		v.BikeID = p.BikeID
	}
	if err = validateLinks(ctx, tx, owner, v.TripID, v.BikeID); err != nil {
		return v, err
	}
	v.Revision++
	v.UpdatedAt = now()
	_, err = tx.ExecContext(ctx, "UPDATE archive_entries SET title=?,trip_id=?,bike_id=?,revision=?,updated_at=? WHERE owner_id=? AND id=?", v.Title, v.TripID, v.BikeID, v.Revision, v.UpdatedAt, owner, id)
	if err != nil {
		return v, err
	}
	if err = recordChange(ctx, tx, &before, v); err != nil {
		return v, err
	}
	return v, tx.Commit()
}
func (s *Store) EntryHistory(ctx context.Context, owner, id string, limit, offset int) ([]domain.EntryChange, error) {
	if _, err := s.Entry(ctx, owner, id); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT revision,origin,actor_id,before_json,after_json,recorded_at FROM entry_changes WHERE owner_id=? AND entry_id=? ORDER BY revision LIMIT ? OFFSET ?", owner, id, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.EntryChange{}
	for rows.Next() {
		var v domain.EntryChange
		var before sql.NullString
		var after string
		if err = rows.Scan(&v.Revision, &v.Origin, &v.ActorID, &before, &after, &v.RecordedAt); err != nil {
			return nil, err
		}
		v.Before = json.RawMessage("null")
		if before.Valid {
			v.Before = json.RawMessage(before.String)
		}
		v.After = json.RawMessage(after)
		items = append(items, v)
	}
	return items, rows.Err()
}

const sourceColumns = "id,owner_id,sha256,byte_length,media_type,created_at"
const sourceSelectColumns = sourceColumns + ", COALESCE((SELECT original_filename FROM source_imports WHERE source_id=sources.id ORDER BY imported_at,id LIMIT 1), '')"

func scanSource(row scanner) (domain.Source, error) {
	var v domain.Source
	err := row.Scan(&v.ID, &v.OwnerID, &v.SHA256, &v.ByteLength, &v.MediaType, &v.CreatedAt, &v.OriginalFilename)
	return v, notFound(err)
}
func (s *Store) PreserveSource(ctx context.Context, owner, filename string, raw []byte) (domain.Source, bool, error) {
	if len(raw) == 0 || len(raw) > 10*1024*1024 || !validName(filename) {
		return domain.Source{}, false, ErrInvalid
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Source{}, false, err
	}
	defer tx.Rollback()
	v, err := scanSource(tx.QueryRowContext(ctx, "SELECT "+sourceSelectColumns+" FROM sources WHERE owner_id=? AND sha256=?", owner, hash))
	duplicate := err == nil
	if errors.Is(err, ErrNotFound) {
		v = domain.Source{OriginalFilename: filename, ID: uuid.NewString(), OwnerID: owner, SHA256: hash, ByteLength: len(raw), MediaType: "application/gpx+xml", CreatedAt: now()}
		_, err = tx.ExecContext(ctx, "INSERT INTO sources ("+sourceColumns+",original) VALUES (?,?,?,?,?,?,?)", v.ID, owner, hash, len(raw), v.MediaType, v.CreatedAt, raw)
	} else if err == nil {
		var previous []byte
		err = tx.QueryRowContext(ctx, "SELECT original FROM sources WHERE id=?", v.ID).Scan(&previous)
		if err == nil && !bytes.Equal(previous, raw) {
			err = errors.New("source hash collision")
		}
	}
	if err != nil {
		return v, false, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO source_imports VALUES (?,?,?,?,?)", uuid.NewString(), owner, v.ID, filename, now())
	if err != nil {
		return v, false, err
	}
	return v, duplicate, tx.Commit()
}
func (s *Store) ListSources(ctx context.Context, owner string, limit, offset int) ([]domain.Source, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+sourceSelectColumns+" FROM sources WHERE owner_id=? ORDER BY created_at,id LIMIT ? OFFSET ?", owner, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Source{}
	for rows.Next() {
		v, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func (s *Store) SourceBytes(ctx context.Context, owner, id string) ([]byte, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, "SELECT original FROM sources WHERE owner_id=? AND id=?", owner, id).Scan(&raw)
	return raw, notFound(err)
}
