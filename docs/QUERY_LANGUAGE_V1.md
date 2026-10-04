# Query language v1: semantics and acceptance corpus

Status: first executable slice for the Ride Archive UI. This contract is intentionally limited to evidence currently stored by the Go/SQLite service; it does not claim to implement natural-language interpretation or derived ride metrics.

## Result unit and available evidence

The result unit is an `ArchiveEntry` (one user-curated outing or supplied route). A result can include zero or more linked GPX paths. A path is displayed as geometry, but its presence does not establish that the user rode it. The current model does not derive entry date, duration, observed distance, elevation gain, riding status, route-vs-recording classification, or a canonical place name. Such values are unknown and cannot be filtered as if they were zero or false.

Searchable text is limited to entry title, trip name, bike name, GPX path name, and original GPX filename. The text mode tokenizes on whitespace, folds case using the browser locale, and requires every token to occur as a substring of at least one searchable value. It does not stem words, interpret units, infer dates, or call a language model. Structured filters combine with AND.

The structured query currently exposed by the page has this logical form:

```json
{
  "version": 1,
  "entity": "archive_entries",
  "where": {
    "all": [
      { "field": "trip_id", "op": "eq", "value": "<trip-id>" },
      { "field": "bike_id", "op": "eq", "value": "<bike-id>" }
    ]
  }
}
```

The UI constructs this query from controls and evaluates it against the loaded archive records. The shape is a versioned semantic draft, not yet a public executor endpoint. `trip_id` and `bike_id` are exact owner-scoped catalog IDs. The structured builder intentionally has no geographic distance filter. When browser geolocation succeeds, results are sorted by minimum great-circle distance from the user location to any stored point in any linked path; this only changes order and never excludes an entry. It is not route length or distance ridden. Entries without geometry have unknown proximity and sort last. Coordinates stay in the browser and are not sent to the server.

The expandable **Query currently applied** panel displays the exact versioned client-side query snapshot used for the visible result set in both text and structured modes. Draft edits do not change that snapshot until the user runs the search. The text predicate shows normalized search terms; structured predicates show the selected catalog IDs.

## Default, ordering, and presentation

On the home page, the initial query includes every archive entry and sorts known distances from the browser location nearest first; proximity does not exclude entries by default. The map centers on the browser location when available, but all entries remain in both map and list results. If location is denied or unavailable, all entries are still shown and distance sorting is omitted. The initial presentation is a map. The same result set can be switched to a list without rerunning the query. The map uses linked GPX geometry and OpenStreetMap tiles; it does not render guessed geometry for an entry without a path.

Text search replaces the default browse query and searches all loaded entries. Structured trip and bike conditions are ANDed. Browse and structured results sort by minimum proximity ascending when it is known, with entries lacking proximity last; this sort is not a filter. Ties preserve archive order. The current UI loads all archive pages and has no separate result pagination. The API list cap is 100,000 records; exceeding it is a future query-service concern.

## Labeled acceptance corpus

The fixture shorthand below assumes entries `Canyon Day 1` (trip `Canyon BDR`, bike `890`, linked GPX `day1.gpx`), `Canyon Day 2` (same trip and bike, linked GPX `day2.gpx`), `Rally Route` (trip `Summer Rally`, no bike, linked GPX `rally.gpx`), and `Unmapped Note` (no trip, no bike, no linked geometry). All fixture locations and names are test data, not assertions about the owner's archive.

| # | User question | Expected interpretation / outcome |
|---:|---|---|
| 1 | Canyon | Text contains `canyon`; returns Day 1 and Day 2. |
| 2 | day 1 | Both tokens required; returns Day 1. |
| 3 | 890 | Text matches bike name; returns Day 1 and Day 2. |
| 4 | GPX rally | Both tokens match searchable values; returns Rally Route. |
| 5 | Canyon BDR | Matches trip name; returns Day 1 and Day 2. |
| 6 | Show me the rally route | Substring search returns Rally Route if all tokens occur across its searchable values; otherwise empty. No semantic paraphrase is inferred. |
| 7 | Trips near me | Unsupported as free text semantics; current proximity ordering does not limit results or group them by trip. |
| 8 | Every ride | Text search for `every` and `ride`; no special “all” operator. Use Reset to return to nearby defaults. |
| 9 | Only Canyon BDR | Structured `trip_id == Canyon BDR`; returns Day 1 and Day 2. |
| 10 | Entries on the Summer Rally trip | Structured `trip_id == Summer Rally`; returns Rally Route. |
| 11 | Entries associated with the 890 | Structured `bike_id == 890`; returns Day 1 and Day 2. Association is user metadata, not proof of which bike recorded a path. |
| 12 | Canyon BDR rides on the 890 | Structured trip and bike equality, ANDed; returns Day 1 and Day 2. |
| 13 | Canyon BDR rides within 50 miles | Unsupported: the structured builder has no distance filter; use the trip filter and inspect proximity ordering. |
| 14 | Everything within 25 miles | Unsupported: distance from the device is never used to exclude entries. |
| 15 | Everything | Clear trip and bike filters; return every entry. |
| 16 | Explore my rides | Browse mode includes all entries; when browser location is available, nearest linked geometry sorts first. |
| 17 | What is the closest ride? | Current UI sorts matching entries by nearest linked point; entries with no geometry sort last. It does not calculate route/ride distance. |
| 18 | Which routes have I ridden? | Clarification required in a future NL interface: supplied path geometry does not prove a ride, and current entries are unclassified. |
| 19 | Rides over 100 miles | Unsupported: observed distance is not currently derived. Do not treat missing distance as zero. |
| 20 | Longest Canyon ride | Unsupported: requires a defined and trustworthy distance metric. |
| 21 | Rides last September | Unsupported: no normalized entry/ride date exists. GPX timestamps remain raw point-level evidence. |
| 22 | Entries with no bike | Could be represented as `bike_id is_null` by a future structured executor; not yet exposed by this builder. |
| 23 | Did I ride the rally route? | Unsupported: an imported route is not evidence of participation. |
| 24 | Rides near Portland on the 890 | Requires place resolution plus bike filtering; place search is not supported. Use device location and the structured bike filter instead. |

## Deferred semantics

This slice does not define a natural-language translator, aliases, entity ambiguity resolution, date/time operators, scalar metric operators, null operators in the UI, aggregates, saved queries, server-side filtering, result snapshots, or a public `POST /query` contract. These require explicit evidence ownership and validation rules before becoming executable. Any later query version must state how unknown values, units, time zones, result limits, sorting, and corrections affect reproducibility.
