"""Expand the read-only sample inventory, keeping user assertions explicit."""
import hashlib
import json
from pathlib import Path
from inspect_gpx import FILES, OUT, inspect

CONTEXT = [
    dict(id='hells', file=FILES[0], bike_id='ktm-890', recorder='onX',
         kind='recorded', trip_id='hells-canyon-2026', partial_day=True),
    dict(id='kittitas', file=FILES[1], bike_id='te-300', recorder='Garmin watch',
         kind='recorded', trip_id=None),
    dict(id='tyee', file='08-tyee-2025.gpx', bike_id='ktm-890', recorder=None,
         kind='route_reference', trip_id='touratech-rally-2025', user_reported_year=2025,
         exact_ride_date=None, user_reports_rode_route=True, exact_path_observed=False),
    dict(id='alder', file='11-alder-creek-2025.gpx', bike_id='ktm-890', recorder=None,
         kind='route_reference', trip_id='touratech-rally-2025', user_reported_year=2025,
         exact_ride_date=None, user_reports_rode_route=True, exact_path_observed=False),
    dict(id='evans', file='evans-300.gpx', bike_id='te-300', recorder=None,
         kind='recorded', trip_id=None, user_location_label='Evans Creek OHV'),
]

def main():
    results=[]
    for context in CONTEXT:
        metrics,_ = inspect(context['file'])
        results.append(dict(id=context['id'],
                            sha256=hashlib.sha256((Path('/Users/dmitryryzhkov/Downloads')/context['file']).read_bytes()).hexdigest(),
                            user_context=context, metrics=metrics))
    (OUT/'corpus.json').write_text(json.dumps(results,indent=2))
    for row in results:
        m=row['metrics']
        print(json.dumps(dict(id=row['id'], points=m['points'], miles=m['distance_miles'],
                              start=m['start_pacific'], end=m['end_pacific'], hours=m['elapsed_hours'],
                              elevation_m=m['elevation_range_m'], max_gap=m['max_interval_seconds'],
                              max_kph=m['max_edge_kph'], missing_time=m['missing_time'],
                              segments=m['segments'], names=m['names'], creator=m['creator'])))

if __name__=='__main__':
    main()
