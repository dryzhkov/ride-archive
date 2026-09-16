"""Read-only GPX sample analysis; no network calls or source edits."""
from pathlib import Path
import collections
import datetime as dt
import json
import math
import statistics
import xml.etree.ElementTree as ET
from zoneinfo import ZoneInfo

OUT = Path(__file__).resolve().parent
NS = {'g': 'http://www.topografix.com/GPX/1/1'}
FILES = ['hells-canyon-day1.gpx', 'kittitas-county-dirt-bike.gpx']

def distance(a, b):
    p, q = math.radians(a['lat']), math.radians(b['lat'])
    dp, dl = q-p, math.radians(b['lon']-a['lon'])
    h = math.sin(dp/2)**2 + math.cos(p)*math.cos(q)*math.sin(dl/2)**2
    return 6371008.8 * 2 * math.asin(min(1, math.sqrt(h)))

def local(t):
    return t.astimezone(ZoneInfo('America/Los_Angeles')).isoformat()

def elevation_change(points, threshold):
    # Reversal/deadband sensitivity diagnostic, not a calibrated ascent algorithm.
    es = [p['ele'] for p in points if p['ele'] is not None]
    if not es:
        return [None, None]
    anchor, up, down = es[0], 0, 0
    for e in es[1:]:
        delta = e-anchor
        if abs(delta) >= threshold:
            up += max(0, delta)
            down += max(0, -delta)
            anchor = e
    return [round(up, 1), round(down, 1)]

def inspect(name):
    path = Path('/Users/dmitryryzhkov/Downloads') / name
    root = ET.parse(path).getroot()
    segs = []
    for seg in root.findall('.//g:trkseg', NS):
        pts = []
        for el in seg.findall('g:trkpt', NS):
            time = el.findtext('g:time', namespaces=NS)
            ele = el.findtext('g:ele', namespaces=NS)
            pts.append(dict(lat=float(el.attrib['lat']), lon=float(el.attrib['lon']),
                            ele=float(ele) if ele else None,
                            time=dt.datetime.fromisoformat(time.replace('Z', '+00:00')) if time else None))
        if pts:
            segs.append(pts)
    pts = [p for seg in segs for p in seg]
    edges = []
    cumulative = 0
    for seg in segs:
        seg[0]['km'] = cumulative/1000
        for a, b in zip(seg, seg[1:]):
            d = distance(a, b)
            seconds = (b['time']-a['time']).total_seconds() if a['time'] and b['time'] else None
            edges.append(dict(m=d, seconds=seconds, kph=d/seconds*3.6 if seconds and seconds>0 else None,
                              start=local(a['time']) if a['time'] else None,
                              end=local(b['time']) if b['time'] else None))
            cumulative += d
            b['km'] = cumulative/1000
    valid = [e for e in edges if e['seconds'] is not None and e['seconds']>0]
    times = [p['time'] for p in pts if p['time']]
    elevations = [p['ele'] for p in pts if p['ele'] is not None]
    result = dict(file=name, creator=root.attrib.get('creator'),
                  names=[t.findtext('g:name', namespaces=NS) for t in root.findall('g:trk', NS)],
                  tracks=len(root.findall('g:trk', NS)), segments=len(segs), points=len(pts),
                  waypoints=len(root.findall('g:wpt', NS)), routes=len(root.findall('g:rte', NS)),
                  fields=collections.Counter(el.tag.split('}')[-1] for p in root.findall('.//g:trkpt', NS) for el in p.iter() if el is not p),
                  missing_time=len(pts)-len(times), missing_elevation=len(pts)-len(elevations),
                  invalid_coordinates=sum(not(-90<=p['lat']<=90 and -180<=p['lon']<=180) for p in pts),
                  nonpositive_time_edges=sum(e['seconds'] is not None and e['seconds']<=0 for e in edges),
                  start_pacific=local(times[0]) if times else None, end_pacific=local(times[-1]) if times else None,
                  elapsed_hours=(times[-1]-times[0]).total_seconds()/3600 if len(times)==len(pts) and times else None,
                  distance_km=cumulative/1000, distance_miles=cumulative/1609.344,
                  start_end_m=distance(pts[0], pts[-1]),
                  bbox=[min(p['lon'] for p in pts), min(p['lat'] for p in pts), max(p['lon'] for p in pts), max(p['lat'] for p in pts)],
                  elevation_range_m=[min(elevations), max(elevations)],
                  median_sample_seconds=statistics.median(e['seconds'] for e in valid) if valid else None,
                  max_interval_seconds=max((e['seconds'] for e in valid), default=None),
                  max_edge_kph=max((e['kph'] for e in valid), default=None),
                  largest_time_intervals=sorted(valid, key=lambda e:e['seconds'], reverse=True)[:10],
                  largest_spatial_steps=sorted(valid, key=lambda e:e['m'], reverse=True)[:5],
                  elevation_gain_loss_sensitivity_m={str(th):[round(sum(elevation_change(seg, th)[i] for seg in segs), 1) for i in [0,1]] for th in [0,3,5,10,20]},
                  gap_sensitivity={str(th):dict(count=sum(e['seconds']>th for e in valid),
                                               minutes=sum(e['seconds'] for e in valid if e['seconds']>th)/60,
                                               connector_km=sum(e['m'] for e in valid if e['seconds']>th)/1000) for th in [30,60,120,300]},
                  moving_hours_sensitivity={str(speed):sum(e['seconds'] for e in valid if e['seconds']<=60 and e['kph']>=speed)/3600 for speed in [1,3,5,8]},
                  spatial_extent_km=[distance({'lat':min(p['lat'] for p in pts),'lon':pts[0]['lon']},{'lat':max(p['lat'] for p in pts),'lon':pts[0]['lon']})/1000,
                                     distance({'lat':statistics.mean(p['lat'] for p in pts),'lon':min(p['lon'] for p in pts)},{'lat':statistics.mean(p['lat'] for p in pts),'lon':max(p['lon'] for p in pts)})/1000])
    result['speed_flag_sensitivity'] = {
        str(limit): dict(count=sum(e['kph']>limit for e in valid),
                        distance_km=sum(e['m'] for e in valid if e['kph']>limit)/1000)
        for limit in [120, 160, 200]
    }
    result['high_point'] = {k: (local(v) if k=='time' else v)
                            for k,v in max(pts, key=lambda p:p['ele']).items() if v is not None}
    result['dwell_candidates_50m_5min'] = []
    for seg in segs:
        if any(p['time'] is None for p in seg):
            continue
        i = 0
        while i < len(seg)-1:
            j = i+1
            while j < len(seg) and distance(seg[i], seg[j]) <= 50:
                j += 1
            seconds = (seg[j-1]['time']-seg[i]['time']).total_seconds()
            if seconds >= 300:
                result['dwell_candidates_50m_5min'].append(dict(
                    start=local(seg[i]['time']), end=local(seg[j-1]['time']),
                    minutes=seconds/60, points=j-i,
                    max_gap_seconds=max((b['time']-a['time']).total_seconds()
                                        for a,b in zip(seg[i:j], seg[i+1:j]))))
                i = j
            else:
                i += 1
    if len(times) != len(pts):
        # Lack of temporal evidence is not evidence of zero gaps, motion or stops.
        for field in ['gap_sensitivity', 'moving_hours_sensitivity',
                      'speed_flag_sensitivity', 'dwell_candidates_50m_5min']:
            result[field] = None
    return result, segs

if __name__ == '__main__':
    data = [inspect(f) for f in FILES]
    (OUT/'metrics.json').write_text(json.dumps([r for r,s in data], indent=2))
    print(json.dumps([r for r,s in data], indent=2))
