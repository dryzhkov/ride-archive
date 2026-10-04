CREATE TABLE users (
 id TEXT PRIMARY KEY,
 display_name TEXT NOT NULL CHECK(length(trim(display_name)) BETWEEN 1 AND 200)
) STRICT;
CREATE TABLE trips (
 id TEXT PRIMARY KEY,
 owner_id TEXT NOT NULL REFERENCES users(id),
 name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 200),
 created_at TEXT NOT NULL,
 UNIQUE(owner_id, id)
) STRICT;
CREATE TABLE bikes (
 id TEXT PRIMARY KEY,
 owner_id TEXT NOT NULL REFERENCES users(id),
 name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 200),
 created_at TEXT NOT NULL,
 UNIQUE(owner_id, id)
) STRICT;
CREATE TABLE archive_entries (
 id TEXT PRIMARY KEY,
 owner_id TEXT NOT NULL REFERENCES users(id),
 title TEXT NOT NULL CHECK(length(trim(title)) BETWEEN 1 AND 200),
 -- Classification becomes available only with the later evidence migration.
 kind TEXT NOT NULL DEFAULT 'unclassified' CHECK(kind = 'unclassified'),
 trip_id TEXT,
 bike_id TEXT,
 revision INTEGER NOT NULL DEFAULT 1 CHECK(revision > 0),
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 UNIQUE(owner_id, id),
 FOREIGN KEY(owner_id, trip_id) REFERENCES trips(owner_id, id),
 FOREIGN KEY(owner_id, bike_id) REFERENCES bikes(owner_id, id)
) STRICT;
CREATE INDEX entries_owner_order ON archive_entries(owner_id, created_at, id);
CREATE INDEX entries_trip ON archive_entries(owner_id, trip_id);
CREATE INDEX entries_bike ON archive_entries(owner_id, bike_id);
CREATE TABLE entry_changes (
 owner_id TEXT NOT NULL,
 entry_id TEXT NOT NULL,
 revision INTEGER NOT NULL CHECK(revision > 0),
 origin TEXT NOT NULL CHECK(origin = 'user'),
 actor_id TEXT NOT NULL REFERENCES users(id),
 before_json TEXT CHECK(before_json IS NULL OR json_valid(before_json)),
 after_json TEXT NOT NULL CHECK(json_valid(after_json)),
 recorded_at TEXT NOT NULL,
 PRIMARY KEY(entry_id, revision),
 FOREIGN KEY(owner_id, entry_id) REFERENCES archive_entries(owner_id, id),
 CHECK(actor_id = owner_id)
) STRICT;
CREATE TRIGGER entry_changes_no_update BEFORE UPDATE ON entry_changes BEGIN
 SELECT RAISE(ABORT, 'entry history is append-only');
END;
CREATE TRIGGER entry_changes_no_delete BEFORE DELETE ON entry_changes BEGIN
 SELECT RAISE(ABORT, 'entry history is append-only');
END;
CREATE TABLE sources (
 id TEXT PRIMARY KEY,
 owner_id TEXT NOT NULL REFERENCES users(id),
 sha256 TEXT NOT NULL CHECK(length(sha256) = 64),
 byte_length INTEGER NOT NULL CHECK(byte_length > 0),
 media_type TEXT NOT NULL,
 original BLOB NOT NULL,
 created_at TEXT NOT NULL,
 CHECK(length(original) = byte_length),
 UNIQUE(owner_id, sha256),
 UNIQUE(owner_id, id)
) STRICT;
CREATE TRIGGER sources_no_update BEFORE UPDATE ON sources BEGIN
 SELECT RAISE(ABORT, 'original sources are immutable');
END;
CREATE TABLE source_imports (
 id TEXT PRIMARY KEY,
 owner_id TEXT NOT NULL,
 source_id TEXT NOT NULL,
 original_filename TEXT NOT NULL,
 imported_at TEXT NOT NULL,
 FOREIGN KEY(owner_id, source_id) REFERENCES sources(owner_id, id)
) STRICT;
CREATE INDEX sources_owner_order ON sources(owner_id, created_at, id);
