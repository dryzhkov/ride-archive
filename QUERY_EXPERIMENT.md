# Ride Archive — query experiment v0

Updated September 15, 2026 · Deterministic query PoC implemented; NL deferred by user choice.

Run instructions and implementation limits: [prototype README](prototype/README.md). SQLite, typed query validation/execution, result-driven map/list, and archive export/restore are implemented. NL acceptance cases and spatial/aggregate operations below remain future work.

## Purpose

Normalize the supplied exports, retrieve correct records with inspectable queries, and export answers independently of the UI. Then test whether NL reliably produces the same query structure. The five-file expansion below adds supplied routes alongside the original recordings.

## Known fixtures

| ID | Bike ID / aliases | Start, Pacific | Raw distance | Recorder / exporter |
|---|---|---|---|---|
| hells | ktm-890 / 890, KTM 890 Adventure R | 2026-09-05 09:46 | 213,328.44 m | onX / onXmaps offroad web |
| kittitas | te-300 / TE 300, Husqvarna TE 300 | 2026-07-25 08:54 | 23,175.91 m | Garmin watch / GaiaGPS |

Query unrounded metrics. Bike/recorder are user assertions; exporter is file metadata; dates are point observations. Hells Canyon is a user-supplied name, not verified geographic enrichment. Source calculations are in analysis/metrics.json and analysis/inspect_gpx.py.

## Example interaction

**Input:** “Show September 2026 recordings on the 890 over 50 miles.”

**Interpretation:** Recorded tracks · KTM 890 Adventure R · start in September 2026, Pacific time · raw recorded distance greater than 50 mi.

**Inspectable/editable query:**

```json
{
  "version": 1,
  "entity": "recordings",
  "where": {
    "all": [
      {"field": "kind", "op": "eq", "value": "recorded"},
      {"field": "bike_id", "op": "eq", "value": "ktm-890"},
      {"field": "started_at", "op": "gte", "value": "2026-09-01T00:00:00-07:00"},
      {"field": "started_at", "op": "lt", "value": "2026-10-01T00:00:00-07:00"},
      {"field": "raw_distance_m", "op": "gt", "value": 80467.2}
    ]
  },
  "order_by": [{"field": "started_at", "direction": "desc"}],
  "limit": 100
}
```

**Expected:** hells. Map displays that track; list explains the matching conditions and retains its data-quality note. It is a raw estimate, not an odometer reading.

JSON is the initial query language. No separate custom text syntax or SQL editor is needed. Direct JSON input and NL translation use the same validator and executor.

## Semantics

- Fields: kind, bike_id, started_at, raw_distance_m, recorder, exporter, has_quality_flags. Operators: eq, in, compatible numeric/time gt/gte/lt/lte, and is_null. AND only initially; reject unsupported OR/NOT.
- Dates mean recording start. Resolve calendar bounds in displayed timezone, store instants in UTC. Relative dates use an explicit reference time; save resolved bounds for reproducible replay. Rolling saved queries are deferred.
- Units: 1 mi = 1,609.344 m. “Over” is strict; “at least” is inclusive.
- Unknown fields do not satisfy ordinary comparisons. Report excluded-unknown counts relevant to the query; is_null finds missing values explicitly.
- Resolve aliases from a user-defined catalog. Clarify non-unique matches rather than guess.
- Cap predicates and result size. Report total match count and truncation; map/list show the same returned set.
- Result envelope: executed query, interpretation, stable IDs, matched fields, relevant provenance/quality, total count, excluded-unknown count and truncation.
- Translation status is ready, needs_clarification or unsupported. Only ready structures passing validation execute.
- Export is a separate explicit action on the executed result/query identity, not a translation side effect.

## Acceptance cases

| Query | Expected |
|---|---|
| All imported recordings | hells + kittitas + evans |
| September 2026, 890, over 50 mi | hells |
| July 2026, TE 300, under 20 mi | kittitas |
| Recorded on Garmin, exported by Gaia | kittitas, with separate provenance |
| Records with anomaly flags | hells, under defined experiment flags |
| TE 300 over 50 mi | Zero matches |
| More than 24 km | hells; correct conversion |
| Unknown motorcycle | Zero here; synthetic missing-bike fixture matches |
| Last month, reference September 14, 2026 Pacific | August 1–September 1; zero |
| Oregon recordings | Unsupported geographic enrichment; do not infer from title |
| Hard trails / actual moving time over 3 hours | Unsupported fields |
| Delete old rides | Unsupported action; read-only interface |

Use explicitly synthetic boundary fixtures separately from real history: exactly 50 mi, just above, month boundaries, missing fields and ambiguous aliases. Test invalid structures, unknown operators and prompt injection. Compare semantic structure, not JSON order or wording.

## Spatial follow-up

Add “recordings intersecting this selected rectangle” after scalar queries work. The map supplies WGS84 geometry; the model does not invent coordinates. Candidate bounding-box filtering must be followed by a defined track/rectangle intersection check. Exclude inferred connections across long sampling gaps from observed-passage claims.

Pick a rectangle around each track and one away from both; inspect expected matches. Define boundary-touch behavior and sparse-sampling uncertainty. Place-name resolution, proximity and overlap are separate later operations.

## Exit criteria

1. Deterministic queries produce exact expected IDs; null, boundary and invalid cases behave as specified.
2. Export/re-import retains original checksums and supported metadata, annotations, observations and queries; result semantics remain equivalent.
3. Structured input works without AI and drives matching map/list results.
4. NL evaluation covers at least 30 labeled cases/paraphrases plus held-out cases. Proposed target: 90% semantic correctness on supported requests, no silent constraint loss in the acceptance set, explicit ambiguity/unsupported paths. Small-set performance is not a general reliability guarantee.
5. Measure translation latency/cost and failure categories. Select provider and data-sharing scope before real model calls.
6. Expand data and ask new user questions before claiming practical or market validation.

## Five-file expansion — September 14

The corpus now contains **three recordings and two rally-supplied routes the user reports riding**. Source inventory and provenance: [corpus findings](analysis/CORPUS_FINDINGS.md) and analysis/corpus.json.

| ID | Kind | Bike | Time evidence | Geometry length |
|---|---|---|---|---|
| tyee | route_reference | ktm-890 | User-reported 2025 rally; no point timestamps | 120.4 mi route length |
| alder | route_reference | ktm-890 | Same multi-day trip; no point timestamps | 28.0 mi route length |
| evans | recorded | te-300 | September 21, 2024, point timestamps | 14.4 mi raw recorded distance |

Contract refinement: add an **archive_entries** entity encompassing both kinds; retain **recordings** as the recorded-only scope used in the example above. Supplied geometry does not become observed geometry because the user rode the route. Keep supplied-route length separate from raw recorded distance, which is null for route references. The geometry-length diagnostic in corpus.json is not yet this normalized database schema.

Add trip_id, user_reported_year and user_reports_rode_route as explicitly sourced query fields. Exact start timestamps remain null on the rally routes. A reported year is not January 1 and is not sufficient for an exact-month query. “My 2025 rides” needs a visible interpretation or clarification about reported participation versus timestamped recordings.

| Additional query | Expected result |
|---|---|
| All archive entries | hells, kittitas, tyee, alder, evans |
| All GPS recordings | hells, kittitas, evans |
| Routes from my 2025 Touratech rally trip | tyee, alder, based on user context |
| Archive entries associated with the 890 | hells, tyee, alder |
| TE 300 recordings | kittitas, evans |
| Recorded in September 2024 | evans |
| Entries without point timestamps | tyee, alder |
| Timestamped recordings in 2025 | No matches; explain two route references have reported 2025 participation but no recorded dates |
| Total recorded distance for the rally | Unknown: route lengths are not observed mileage; do not return zero or 148.4 mi |
| How often did I ride Evans Creek? | One imported recording; actual visit count unknown. User says they rode there often. |

Default aggregate semantics must distinguish “no observed values” from a measured zero. Do not label supplied-route lengths as riding totals. These real missing-data and provenance cases replace the need to fabricate every null fixture; numeric/date boundary fixtures are still necessary.
