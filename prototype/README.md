# Ride Archive query PoC

Run from the project root with Python 3 (standard library only):

```sh
python3 -m ride_archive.server --port 8766
```

Open http://127.0.0.1:8766. First startup seeds `local_data/archive.sqlite3` from the five existing GPX fixtures, their verified analysis, and user-provided context. Original files must still be in Downloads for initial seeding. Subsequent starts reuse the database. This is a fixture importer, not a general upload pipeline.

Choose a structured-query example, inspect/edit its JSON, and run it. The API validates the contract, compiles parameterized read-only SQL, and returns matching entries, provenance, total counts, unknown counts, and truncation. Map and list use the returned records. Invalid queries retain the previous results with an error. Natural-language translation is deliberately unconfigured; the examples are structured queries, not simulated AI.

The version 1 query supports AND predicates, typed scalar comparisons, explicit null matching, sorting, and limits. See `ride_archive/query.py` for the field/operator contract, `/api/catalog` for the catalog, and `QUERY_EXPERIMENT.md` for semantics. There is no arbitrary SQL, geographic search, aggregation, or data editing yet. Dates are recording-start instants with explicit timezone offsets. Distances use meters; supplied-route length and observed recording distance are separate fields.

## Export and restore

Export the entire archive or only the shown results. ZIP bundles contain a JSON manifest, metadata/provenance, display geometry, and checksum-verified original GPX files. Result exports also preserve the executed structured query. Saved queries reside in this browser's local storage and are not included in full archive exports.

Restore into a new database (existing destinations are rejected):

```sh
python3 -m ride_archive.server --restore /path/to/ride-archive.zip --db local_data/restored.sqlite3 --init-only
python3 -m ride_archive.server --db local_data/restored.sqlite3 --port 8767
```

The manifest's executed query can be pasted into the editor; restoring does not import browser-saved queries. Original GPX bytes retain full precision; display geometry uses the existing rounded prototype representation.

## Validation

```sh
python3 -m unittest discover -s tests -v
```

Tests cover the five fixtures, null semantics, exact distance/date boundaries, schema rejection, SQL parameterization, limits, read-only execution, and export/restore fidelity.

The server binds to loopback and is a local PoC, not a hosted service. Leaflet and OpenStreetMap require internet; tile requests reveal the viewed region. Coordinates remain local. Raw distance includes sampling-gap connectors and is not an odometer reading. See `analysis/FINDINGS.md` and `analysis/CORPUS_FINDINGS.md` for data limitations.
