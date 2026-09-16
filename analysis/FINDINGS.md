# Two-file feasibility check

Analysis of the supplied exports; originals unchanged. This is an exploratory data check, not production ingestion code or a calibrated GPS correction pipeline. Reproduce with `python3 analysis/inspect_gpx.py`; computed detail is in `metrics.json`.

## What was supplied versus what was discovered

| | Hells Canyon | Kittitas County |
|---|---|---|
| User context | Part of day 1 of a two-day ADV trip, KTM 890 Adventure R, recorded on onX | Local mountain enduro outing, TE 300, recorded on Garmin watch |
| GPX creator | onXmaps offroad web | GaiaGPS |
| Embedded name | Track 09/05/26 16:45 | Kittitas County Dirt Bike |
| Tracks / segments / points | 1 / 1 / 4,737 | 1 / 1 / 6,034 |
| Timestamp/elevation coverage | 100% / 100% | 100% / 100% |
| Named waypoints / routes | 0 / 0 | 0 / 0 |

The Kittitas creator is compatible with a Garmin recording passing through Gaia, but the file does not independently establish that chain. Keep recorder, exporter and user-confirmed bike as separate provenance. Neither file contains bike identity, trip membership, heart rate, terrain classification or meaningful narrative. Point children contain only time and elevation beyond the latitude/longitude attributes.

## Baseline results

All local times below explicitly use America/Los_Angeles (PDT); GPX times carry UTC. Local display timezone is our choice, not embedded ride metadata.

| Measurement | Hells Canyon recording | Kittitas recording |
|---|---|---|
| Date | September 5, 2026 | July 25, 2026 |
| Timestamp span | 09:46–15:41 PDT; 5 h 55 min | 08:54–14:06 PDT; 5 h 12 min |
| Raw point-to-point distance | 132.6 mi / 213.3 km | 14.4 mi / 23.2 km |
| Recorded elevation range | 497–2,424 m / roughly 1,630–7,953 ft | 789–1,957 m / roughly 2,590–6,421 ft |
| Start/end straight-line separation | 65.0 km / 40.4 mi | 13.2 km / 8.2 mi |
| Median sampling interval | 3.001 seconds | 2 seconds |
| Longest interval | 23 min 21 sec | 26 seconds |

These distances sum spherical great-circle distances between adjacent points within the single segment, using mean Earth radius 6,371,008.8 m. They are not ground truth or corrected odometer readings. Raw totals include connectors across time gaps. Neither export contains invalid coordinate ranges, missing point timestamps/elevations, or non-increasing point times.

## Findings that matter

1. **onX speed/time quality needs investigation.** The largest adjacent jump covers 377 m in about three seconds, implying 451 km/h (280 mph). There are 34 intervals above 200 km/h. These may reflect location/timestamp alignment, missing samples or other export issues; the file alone does not establish the cause. Some jumps continue forward rather than spiking out and back, so deleting fast intervals would also delete plausible traveled distance. Do not claim a corrected distance or maximum speed.
2. **Long ADV intervals look like possible pauses, but are unobserved time.** Five intervals exceed five minutes, totaling 67.7 minutes; endpoints for those intervals are only about 18–88 m apart. This is consistent with breaks or paused recording, but cannot exclude movement during the gap. The longest is 12:57–13:20 PDT, endpoints about 30 m apart. All intervals over 60 seconds total 105.1 minutes, with only 0.67 km of straight connectors; long-gap connectors are not the main raw-distance contribution.
3. **Enduro has substantial time with little geographic progress.** A simple detector finds 18 non-overlapping candidate windows totaling about 153 minutes with all sampled positions within 50 m of each window's anchor and duration at least five minutes. The longest sampling gap is only 26 seconds. These are dwell candidates, not 18 confirmed stops: one stop can fragment into several windows, and technical maneuvers count as movement within an area.
4. **Moving time is a policy-dependent estimate.** Summing intervals of at most 60 seconds whose point-to-point speed exceeds 1, 3, or 5 km/h gives approximately 1 h 58 min, 1 h 30 min, or 1 h 19 min for enduro. That 39-minute change is too large to hide behind a single precise number. The same thresholds give roughly 4 h 10 min, 4 h 6 min, and 4 h 2 min for onX, but unreliable local speeds and unobserved intervals still limit trust.
5. **Both records have coherent high points.** Hells Canyon reaches a recorded 2,424 m around 13:44 PDT; Kittitas reaches 1,957 m around 13:38 PDT. Neighboring elevation samples support these peaks; that does not independently validate altitude accuracy. The onX minimum has an isolated dip, so round elevation ranges rather than overstate precision.
6. **Accumulated climb is sensitive to processing.** Raw positive elevation deltas total 6,013 m for onX and 1,999 m for Kittitas. A simple 5–10 m deadband lowers those to 4,570–4,999 m and 1,623–1,722 m. These are algorithm-sensitivity examples, not validated gain estimates or confidence bounds. Keep elevation profiles; postpone headline climb totals until compared with original recorder summaries.
7. **An outing is not necessarily a closed loop or a whole day.** Hells Canyon is explicitly partial from your description. Kittitas ends 8.2 straight-line miles from its start and about 745 m higher. That may be intentional or incomplete; GPX alone cannot decide. The app needs to preserve a recording without presenting it as a verified complete outing.

## What is possible without AI

| Capability | Evidence from these files | Remaining requirement |
|---|---|---|
| Automatic dated archive and chronological view | Complete timestamps on both | Explicit timezone and editable fields |
| Combined map, bike/date filters | Complete geometry; bikes supplied by user | Store small amount of user context |
| Search by map area / find prior visits | Coordinates sufficient | Spatial indexing; more recordings to validate repeat visits |
| Elevation profile and approximate extent | Complete elevation/geometry | Round values and expose quality limitations |
| Trip grouping | Recordings can be grouped | User confirms membership/order and missing portions |
| Photo alignment later | Time axis available | Photo timestamps/timezone; no interpolation across long unobserved gaps |
| Heatmap later | Geometry available | Count distinct rides or coverage, not points; recordings have different sampling/stationary behavior |

## What needs enrichment or user input

- Broad place names need geographic lookup unless present in user/track text. “Hells Canyon” comes from your description/filename here, not the onX track title. No geographic lookup was performed and no tracks were uploaded to a mapping or AI provider.
- Trail/road names need geographic data and conservative matching; nearby geometry is not proof a trail was ridden. Neither file contains a ready-made trail itinerary.
- Motorcycle, ADV/enduro designation and two-day trip structure come from you. They cannot be recovered reliably from speed or geometry.
- AI can translate a question into filters or summarize verified facts later. It cannot repair missing observations or infer why a ride stopped. Named-place search still requires dependable names/aliases under the model.

## Feasibility conclusion

**These two files support a useful deterministic archive baseline, but do not yet establish product retention or accuracy across exporters.** Both preserve enough structure for dated map records, elevation profiles, browsing and geographic retrieval. The strongest value beyond storage is making a collection intelligible across recordings, bikes, places and trips. Individual distance/elevation statistics are useful context but not a distinctive product by themselves.

Next verification should compare the onX distance/duration with its original activity summary and confirm whether Kittitas ends where you intended. That distinguishes source-data limitations from analysis mistakes before choosing headline metrics. No application architecture or MVP approval is implied by this analysis.
