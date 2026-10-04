package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"ridearchive/internal/domain"
	"ridearchive/internal/gpx"
)

const pathSummaryColumns = `p.id,sp.source_id,
 COALESCE((SELECT original_filename FROM source_imports WHERE source_id=sp.source_id ORDER BY imported_at,id LIMIT 1),''),
 p.name,p.structure,p.source_locator,p.point_count,p.segment_count`

func scanPaths(rows *sql.Rows) ([]domain.PathSummary, error) {
	defer rows.Close()
	out := []domain.PathSummary{}
	for rows.Next() {
		var p domain.PathSummary
		if err := rows.Scan(&p.ID, &p.SourceID, &p.OriginalFilename, &p.Name, &p.Structure, &p.SourceLocator, &p.PointCount, &p.SegmentCount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *Store) SourcePaths(ctx context.Context, owner, sourceID string) ([]domain.PathSummary, error) {
	var found int
	if err := s.db.QueryRowContext(ctx, "SELECT 1 FROM sources WHERE owner_id=? AND id=?", owner, sourceID).Scan(&found); err != nil {
		return nil, notFound(err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+pathSummaryColumns+` FROM paths p JOIN source_parses sp ON sp.id=p.parse_id
 WHERE p.owner_id=? AND sp.source_id=? AND sp.parser_version=? ORDER BY p.source_locator`, owner, sourceID, gpx.Version)
	if err != nil {
		return nil, err
	}
	return scanPaths(rows)
}
func (s *Store) ParseSource(ctx context.Context, owner, sourceID string) ([]domain.PathSummary, error) {
	// Parse outside the write transaction; stored bytes are immutable. The unique
	// parse constraint plus serialized transaction makes concurrent retries safe.
	var existing int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM source_parses WHERE owner_id=? AND source_id=? AND parser_version=?", owner, sourceID, gpx.Version).Scan(&existing); err != nil {
		return nil, err
	}
	if existing > 0 {
		return s.SourcePaths(ctx, owner, sourceID)
	}
	raw, err := s.SourceBytes(ctx, owner, sourceID)
	if err != nil {
		return nil, err
	}
	parsed, err := gpx.Parse(raw)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var parseID string
	err = tx.QueryRowContext(ctx, "SELECT id FROM source_parses WHERE owner_id=? AND source_id=? AND parser_version=?", owner, sourceID, gpx.Version).Scan(&parseID)
	if errors.Is(err, sql.ErrNoRows) {
		parseID = uuid.NewString()
		if _, err = tx.ExecContext(ctx, "INSERT INTO source_parses VALUES (?,?,?,?,?)", parseID, owner, sourceID, gpx.Version, now()); err != nil {
			return nil, err
		}
		for _, p := range parsed {
			geometry, err := json.Marshal(p.Segments)
			if err != nil {
				return nil, err
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO paths VALUES (?,?,?,?,?,?,?,?,?)", uuid.NewString(), owner, parseID, p.Name, p.Structure, p.Locator, p.PointCount, len(p.Segments), string(geometry)); err != nil {
				return nil, err
			}
		}
	} else if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.SourcePaths(ctx, owner, sourceID)
}
func (s *Store) EntryPaths(ctx context.Context, owner, entryID string) ([]domain.PathSummary, error) {
	if _, err := s.Entry(ctx, owner, entryID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+pathSummaryColumns+` FROM entry_paths ep JOIN paths p ON p.id=ep.path_id JOIN source_parses sp ON sp.id=p.parse_id
 WHERE ep.owner_id=? AND ep.entry_id=? ORDER BY ep.linked_at,p.id`, owner, entryID)
	if err != nil {
		return nil, err
	}
	return scanPaths(rows)
}

func (s *Store) EntryMapPaths(ctx context.Context, owner, entryID string) ([]domain.PathGeometry, error) {
	if _, err := s.Entry(ctx, owner, entryID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+pathSummaryColumns+`,p.geometry_json FROM entry_paths ep
JOIN paths p ON p.id=ep.path_id JOIN source_parses sp ON sp.id=p.parse_id
WHERE ep.owner_id=? AND ep.entry_id=? ORDER BY ep.linked_at,p.id`, owner, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PathGeometry{}
	for rows.Next() {
		var p domain.PathGeometry
		var raw string
		if err := rows.Scan(&p.ID, &p.SourceID, &p.OriginalFilename, &p.Name, &p.Structure, &p.SourceLocator, &p.PointCount, &p.SegmentCount, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &p.Segments); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
func (s *Store) SetPathLink(ctx context.Context, owner, entryID, pathID string, revision int, attach bool) (domain.ArchiveEntry, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ArchiveEntry{}, err
	}
	defer tx.Rollback()
	e, err := scanEntry(tx.QueryRowContext(ctx, "SELECT "+entryColumns+" FROM archive_entries WHERE owner_id=? AND id=?", owner, entryID))
	if err != nil {
		return e, err
	}
	if e.Revision != revision {
		return e, ErrConflict
	}
	var pointCount int
	if err = tx.QueryRowContext(ctx, "SELECT point_count FROM paths WHERE owner_id=? AND id=?", owner, pathID).Scan(&pointCount); err != nil {
		return e, notFound(err)
	}
	if pointCount == 0 {
		return e, ErrInvalid
	}
	var present int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM entry_paths WHERE entry_id=? AND path_id=?", entryID, pathID).Scan(&present); err != nil {
		return e, err
	}
	if (present > 0) == attach {
		return e, nil
	}
	before := e
	e.Revision++
	e.UpdatedAt = now()
	action := "detached"
	if attach {
		action = "attached"
		_, err = tx.ExecContext(ctx, "INSERT INTO entry_paths VALUES (?,?,?,?)", owner, entryID, pathID, e.UpdatedAt)
	} else {
		_, err = tx.ExecContext(ctx, "DELETE FROM entry_paths WHERE owner_id=? AND entry_id=? AND path_id=?", owner, entryID, pathID)
	}
	if err != nil {
		return e, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE archive_entries SET revision=?,updated_at=? WHERE owner_id=? AND id=?", e.Revision, e.UpdatedAt, owner, entryID); err != nil {
		return e, err
	}
	if err = recordChange(ctx, tx, &before, e); err != nil {
		return e, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO path_link_changes VALUES (?,?,?,?,?,?,?)", owner, entryID, e.Revision, pathID, action, owner, e.UpdatedAt); err != nil {
		return e, err
	}
	return e, tx.Commit()
}

// SaveEntryWithGPX atomically creates an entry, preserves/parses the original,
// and attaches every non-empty track/route in it. If entryID is set, it adds a
// file to that existing entry and requires its current revision.
func (s *Store) SaveEntryWithGPX(ctx context.Context, owner, entryID, title string, trip, bike *string, revision int, filename string, raw []byte) (domain.ArchiveEntry, domain.Source, []domain.PathSummary, error) {
	var zero domain.ArchiveEntry
	if len(raw) == 0 || len(raw) > 10*1024*1024 || !validName(filename) {
		return zero, domain.Source{}, nil, ErrInvalid
	}
	parsed, err := gpx.Parse(raw)
	if err != nil {
		return zero, domain.Source{}, nil, err
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return zero, domain.Source{}, nil, err
	}
	defer tx.Rollback()
	var e domain.ArchiveEntry
	if entryID == "" {
		if !validName(title) {
			return zero, domain.Source{}, nil, ErrInvalid
		}
		if err = validateLinks(ctx, tx, owner, trip, bike); err != nil {
			return zero, domain.Source{}, nil, err
		}
		stamp := now()
		e = domain.ArchiveEntry{ID: uuid.NewString(), OwnerID: owner, Title: strings.TrimSpace(title), Kind: domain.Unclassified, TripID: trip, BikeID: bike, Revision: 1, CreatedAt: stamp, UpdatedAt: stamp}
		if _, err = tx.ExecContext(ctx, "INSERT INTO archive_entries ("+entryColumns+") VALUES (?,?,?,?,?,?,?,?,?)", e.ID, e.OwnerID, e.Title, e.Kind, e.TripID, e.BikeID, e.Revision, e.CreatedAt, e.UpdatedAt); err != nil {
			return zero, domain.Source{}, nil, err
		}
		if err = recordChange(ctx, tx, nil, e); err != nil {
			return zero, domain.Source{}, nil, err
		}
	} else {
		e, err = scanEntry(tx.QueryRowContext(ctx, "SELECT "+entryColumns+" FROM archive_entries WHERE owner_id=? AND id=?", owner, entryID))
		if err != nil {
			return zero, domain.Source{}, nil, err
		}
		if e.Revision != revision {
			return e, domain.Source{}, nil, ErrConflict
		}
	}
	src, err := sourceForImportTx(ctx, tx, owner, filename, raw, hash)
	if err != nil {
		return e, domain.Source{}, nil, err
	}
	var parseID string
	err = tx.QueryRowContext(ctx, "SELECT id FROM source_parses WHERE owner_id=? AND source_id=? AND parser_version=?", owner, src.ID, gpx.Version).Scan(&parseID)
	if errors.Is(err, sql.ErrNoRows) {
		parseID = uuid.NewString()
		if _, err = tx.ExecContext(ctx, "INSERT INTO source_parses VALUES (?,?,?,?,?)", parseID, owner, src.ID, gpx.Version, now()); err != nil {
			return e, src, nil, err
		}
		for _, p := range parsed {
			geometry, merr := json.Marshal(p.Segments)
			if merr != nil {
				return e, src, nil, merr
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO paths VALUES (?,?,?,?,?,?,?,?,?)", uuid.NewString(), owner, parseID, p.Name, p.Structure, p.Locator, p.PointCount, len(p.Segments), string(geometry)); err != nil {
				return e, src, nil, err
			}
		}
	} else if err != nil {
		return e, src, nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT p.id FROM paths p WHERE p.owner_id=? AND p.parse_id=? AND p.point_count>0 ORDER BY p.source_locator`, owner, parseID)
	if err != nil {
		return e, src, nil, err
	}
	pathIDs := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return e, src, nil, err
		}
		pathIDs = append(pathIDs, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return e, src, nil, err
	}
	if len(pathIDs) == 0 {
		return e, src, nil, ErrInvalid
	}
	for _, pathID := range pathIDs {
		if err = attachPathTx(ctx, tx, owner, &e, pathID); err != nil {
			return e, src, nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return e, src, nil, err
	}
	paths, err := s.EntryPaths(ctx, owner, e.ID)
	if err != nil {
		return e, src, nil, err
	}
	current := make([]domain.PathSummary, 0, len(paths))
	for _, path := range paths {
		if path.SourceID == src.ID {
			current = append(current, path)
		}
	}
	return e, src, current, nil
}

func sourceForImportTx(ctx context.Context, tx *sql.Tx, owner, filename string, raw []byte, hash string) (domain.Source, error) {
	var src domain.Source
	err := tx.QueryRowContext(ctx, "SELECT "+sourceSelectColumns+" FROM sources WHERE owner_id=? AND sha256=?", owner, hash).Scan(&src.ID, &src.OwnerID, &src.SHA256, &src.ByteLength, &src.MediaType, &src.CreatedAt, &src.OriginalFilename)
	if errors.Is(err, sql.ErrNoRows) {
		src = domain.Source{ID: uuid.NewString(), OwnerID: owner, SHA256: hash, ByteLength: len(raw), MediaType: "application/gpx+xml", CreatedAt: now(), OriginalFilename: filename}
		_, err = tx.ExecContext(ctx, "INSERT INTO sources ("+sourceColumns+",original) VALUES (?,?,?,?,?,?,?)", src.ID, owner, hash, len(raw), src.MediaType, src.CreatedAt, raw)
	} else if err == nil {
		var prior []byte
		err = tx.QueryRowContext(ctx, "SELECT original FROM sources WHERE owner_id=? AND id=?", owner, src.ID).Scan(&prior)
		if err == nil && !bytes.Equal(prior, raw) {
			err = errors.New("source hash collision")
		}
	}
	if err != nil {
		return src, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO source_imports VALUES (?,?,?,?,?)", uuid.NewString(), owner, src.ID, filename, now())
	return src, err
}
func attachPathTx(ctx context.Context, tx *sql.Tx, owner string, e *domain.ArchiveEntry, pathID string) error {
	var points int
	if err := tx.QueryRowContext(ctx, "SELECT point_count FROM paths WHERE owner_id=? AND id=?", owner, pathID).Scan(&points); err != nil {
		return notFound(err)
	}
	if points == 0 {
		return ErrInvalid
	}
	var present int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM entry_paths WHERE entry_id=? AND path_id=?", e.ID, pathID).Scan(&present); err != nil {
		return err
	}
	if present > 0 {
		return nil
	}
	before := *e
	stamp := now()
	if _, err := tx.ExecContext(ctx, "INSERT INTO entry_paths VALUES (?,?,?,?)", owner, e.ID, pathID, stamp); err != nil {
		return err
	}
	e.Revision++
	e.UpdatedAt = stamp
	if _, err := tx.ExecContext(ctx, "UPDATE archive_entries SET revision=?,updated_at=? WHERE owner_id=? AND id=?", e.Revision, stamp, owner, e.ID); err != nil {
		return err
	}
	if err := recordChange(ctx, tx, &before, *e); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO path_link_changes VALUES (?,?,?,?,?,?,?)", owner, e.ID, e.Revision, pathID, "attached", owner, stamp)
	return err
}
