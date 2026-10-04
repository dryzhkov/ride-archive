-- Additive migration: original files and existing metadata remain unchanged.
CREATE TABLE source_parses (
 id TEXT PRIMARY KEY,
 owner_id TEXT NOT NULL,
 source_id TEXT NOT NULL,
 parser_version TEXT NOT NULL,
 parsed_at TEXT NOT NULL,
 UNIQUE(owner_id, source_id, parser_version),
 UNIQUE(owner_id, id),
 FOREIGN KEY(owner_id, source_id) REFERENCES sources(owner_id, id)
) STRICT;
CREATE TABLE paths (
 id TEXT PRIMARY KEY,
 owner_id TEXT NOT NULL,
 parse_id TEXT NOT NULL,
 name TEXT NOT NULL,
 structure TEXT NOT NULL CHECK(structure IN ('gpx_track','gpx_route')),
 source_locator TEXT NOT NULL,
 point_count INTEGER NOT NULL CHECK(point_count >= 0),
 segment_count INTEGER NOT NULL CHECK(segment_count >= 0),
 geometry_json TEXT NOT NULL CHECK(json_valid(geometry_json)),
 UNIQUE(parse_id, source_locator),
 UNIQUE(owner_id, id),
 FOREIGN KEY(owner_id, parse_id) REFERENCES source_parses(owner_id, id)
) STRICT;
CREATE TABLE entry_paths (
 owner_id TEXT NOT NULL,
 entry_id TEXT NOT NULL,
 path_id TEXT NOT NULL,
 linked_at TEXT NOT NULL,
 PRIMARY KEY(entry_id,path_id),
 FOREIGN KEY(owner_id,entry_id) REFERENCES archive_entries(owner_id,id),
 FOREIGN KEY(owner_id,path_id) REFERENCES paths(owner_id,id)
) STRICT;
CREATE TABLE path_link_changes (
 owner_id TEXT NOT NULL,
 entry_id TEXT NOT NULL,
 revision INTEGER NOT NULL,
 path_id TEXT NOT NULL,
 action TEXT NOT NULL CHECK(action IN ('attached','detached')),
 actor_id TEXT NOT NULL REFERENCES users(id),
 recorded_at TEXT NOT NULL,
 PRIMARY KEY(entry_id,revision),
 FOREIGN KEY(owner_id,entry_id) REFERENCES archive_entries(owner_id,id),
 FOREIGN KEY(owner_id,path_id) REFERENCES paths(owner_id,id),
 CHECK(actor_id=owner_id)
) STRICT;
CREATE TRIGGER path_link_changes_no_update BEFORE UPDATE ON path_link_changes BEGIN
 SELECT RAISE(ABORT, 'link history is append-only');
END;
CREATE TRIGGER path_link_changes_no_delete BEFORE DELETE ON path_link_changes BEGIN
 SELECT RAISE(ABORT, 'link history is append-only');
END;
