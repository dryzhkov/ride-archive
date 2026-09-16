# Ride Archive — expanded sample inventory

September 14, 2026 · Five files inspected locally; originals unchanged

| Entry | Source geometry | Bike, user-supplied | Date evidence | Geometry length |
|---|---|---|---|---|
| Hells Canyon | GPS recording | KTM 890 Adventure R | September 5, 2026 | 132.6 mi raw recorded |
| Kittitas County | GPS recording | TE 300 | July 25, 2026 | 14.4 mi raw recorded |
| Tyee | Rally-supplied route, user confirms riding it | KTM 890 Adventure R | 2025, user-reported; exact date unknown | 120.4 mi supplied route |
| Alder Creek | Rally-supplied route, user confirms riding it | KTM 890 Adventure R | Same 2025 rally trip; different days, exact dates/order unknown | 28.0 mi supplied route |
| Evans Creek | GPS recording | TE 300 | September 21, 2024 | 14.4 mi raw recorded |

Lengths use adjacent coordinate distances within segments, not corrected odometer measurements. Do not add route lengths to riding mileage.

## New export facts

- **Tyee:** 8,866 points, one track/segment, elevations on every point, no timestamps. Embedded name “08 - Tyee 2025”; exporter GaiaGPS.
- **Alder Creek:** 1,616 points, one track/segment, elevations on every point, no timestamps. Embedded name “11 - Alder Creek 2025”; exporter GaiaGPS.
- **Evans:** 4,989 points, one track/segment, complete timestamps/elevations. September 21, 2024, 08:00–12:10 Pacific; elapsed span about 4 h 11 min. Embedded name “Pierce County Dirt Bike”; exporter GaiaGPS. Original recorder is unconfirmed. Longest sample interval 21 seconds; maximum adjacent-point speed estimate about 110 km/h is not a verified maximum riding speed.
- Filenames/track prefixes 08 and 11 are not evidence of month, date or day order.

## User assertions retained

Tyee and Alder Creek were supplied rally routes ridden on the 890 during multiple days of the same 2025 Touratech rally trip. This is an association with routes, not proof of exact-path completion. Exact dates, deviations and measured duration/mileage remain unknown.

Evans Creek is a familiar local OHV area the user rode often on the 300. This corpus contains only one Evans recording; actual visit count cannot be inferred. Its user location label is richer than the embedded county-level title and remains attributed to the user, not geocoding.

## Implications for the query experiment

1. Store both route references and recordings; keep an explicit recorded-only query scope.
2. Keep observed start time, user-reported year, trip association and bike as distinct fields with provenance.
3. Queries for trip participation can match the rally routes; queries for measured 2025 riding duration cannot.
4. Missing dates/durations are null, not zero or fabricated January dates. No timestamp-based stop/speed results for route references.
5. Five entries already exercise bike, trip, date, source, kind and unknown-data queries. More files are not required before the deterministic experiment.

Reproduce with `python3 analysis/inspect_corpus.py`. See corpus.json for per-file hashes, user context and computed diagnostics. The existing web prototype remains unchanged at two recordings.
