"""Local SQLite storage, original preservation and portable archive bundles."""
import hashlib
import io
import json
from pathlib import Path
import sqlite3
import zipfile
from .query import FIELDS, value_for

ROOT=Path(__file__).resolve().parent.parent
DATA=ROOT/'local_data'
DB=DATA/'archive.sqlite3'
SCHEMA='''
CREATE TABLE IF NOT EXISTS sources(id TEXT PRIMARY KEY, filename TEXT NOT NULL, sha256 TEXT NOT NULL UNIQUE, original BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS entries(
 id TEXT PRIMARY KEY REFERENCES sources(id), kind TEXT NOT NULL, bike_id TEXT, trip_id TEXT,
 recorder TEXT, exporter TEXT, started_at TEXT, raw_distance_m REAL, route_length_m REAL,
 user_reported_year INTEGER, user_reports_rode_route INTEGER, has_quality_flags INTEGER,
 provenance TEXT NOT NULL, display TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS entry_bike_date ON entries(bike_id,started_at);
PRAGMA user_version=1;
'''

def connect(path=DB, readonly=False):
    if readonly:
        conn=sqlite3.connect(f'{Path(path).resolve().as_uri()}?mode=ro',uri=True)
        conn.execute('PRAGMA query_only=ON')
    else:
        Path(path).parent.mkdir(parents=True,exist_ok=True)
        conn=sqlite3.connect(path)
    conn.row_factory=sqlite3.Row
    conn.execute('PRAGMA foreign_keys=ON')
    return conn

def seed(path=DB):
    corpus=json.loads((ROOT/'analysis/corpus.json').read_text())
    displays={r['id']:r for r in json.loads((ROOT/'prototype/rides.json').read_text())}
    with connect(path) as conn:
        conn.executescript(SCHEMA)
        for record in corpus:
            ident=record['id']; c=record['user_context']; m=record['metrics']
            if conn.execute('SELECT 1 FROM entries WHERE id=?',(ident,)).fetchone(): continue
            original=(Path('/Users/dmitryryzhkov/Downloads')/c['file']).read_bytes()
            if hashlib.sha256(original).hexdigest()!=record['sha256']: raise ValueError(f'Source changed: {c["file"]}')
            route=c['kind']=='route_reference'
            values=dict(id=ident,kind=c['kind'],bike_id=c['bike_id'],trip_id=c.get('trip_id'),recorder=c.get('recorder'),
                        exporter=m['creator'],started_at=value_for('started_at',m['start_pacific']) if m['start_pacific'] else None,
                        raw_distance_m=None if route else m['distance_km']*1000,route_length_m=m['distance_km']*1000 if route else None,
                        user_reported_year=c.get('user_reported_year'),user_reports_rode_route=c.get('user_reports_rode_route'),
                        has_quality_flags=None if route else bool(m['max_edge_kph'] and m['max_edge_kph']>200))
            provenance=dict(user_context=c,source={'filename':c['file'],'exporter':m['creator'],'sha256':record['sha256']},
                            metrics={'method':'adjacent-haversine-mean-earth-v1','distance_includes_gap_connectors':True,
                                     'quality_flag_method':'adjacent-speed-over-200-kph-v1','timestamps_missing':m['missing_time']})
            values.update(provenance=json.dumps(provenance),display=json.dumps(displays[ident]))
            conn.execute('INSERT INTO sources VALUES (?,?,?,?)',(ident,c['file'],record['sha256'],original))
            conn.execute(f'INSERT INTO entries ({",".join(values)}) VALUES ({",".join("?" for _ in values)})',list(values.values()))

def export_bundle(conn, ids=None, query=None):
    entries=[dict(r) for r in conn.execute('SELECT * FROM entries ORDER BY id') if ids is None or r['id'] in ids]
    sources=[]; stream=io.BytesIO()
    with zipfile.ZipFile(stream,'w',zipfile.ZIP_DEFLATED) as z:
        for entry in entries:
            source=dict(conn.execute('SELECT * FROM sources WHERE id=?',(entry['id'],)).fetchone())
            original=source.pop('original'); member=f'originals/{source["sha256"]}.gpx'
            source['member']=member;sources.append(source);z.writestr(member,original)
            entry['provenance']=json.loads(entry['provenance']);entry['display']=json.loads(entry['display'])
        z.writestr('manifest.json',json.dumps(dict(format='ride-archive',version=1,entries=entries,sources=sources,executed_query=query),indent=2))
    return stream.getvalue()

def restore_bundle(bundle, path):
    # Never extract ZIP paths. Restore to a new file only, transactionally.
    path=Path(path)
    if path.exists(): raise ValueError('Restore destination must not exist.')
    with zipfile.ZipFile(io.BytesIO(bundle)) as z:
        if sum(i.file_size for i in z.infolist())>50_000_000: raise ValueError('Bundle exceeds 50 MB limit.')
        manifest=json.loads(z.read('manifest.json'))
        if manifest.get('format')!='ride-archive' or manifest.get('version')!=1: raise ValueError('Unsupported bundle.')
        sources=[]
        for source in manifest['sources']:
            raw=z.read(source['member'])
            if hashlib.sha256(raw).hexdigest()!=source['sha256']: raise ValueError('Original checksum mismatch.')
            sources.append((source['id'],source['filename'],source['sha256'],raw))
        entries=[]
        columns=[*FIELDS,'provenance','display']
        for entry in manifest['entries']:
            if set(entry)!=set(columns): raise ValueError('Unexpected entry fields.')
            for f in FIELDS:
                if entry[f] is not None:
                    value=bool(entry[f]) if FIELDS[f]=='boolean' and entry[f] in (0,1) else entry[f]
                    entry[f]=value_for(f,value)
            entry['provenance']=json.dumps(entry['provenance']);entry['display']=json.dumps(entry['display'])
            entries.append([entry[c] for c in columns])
    with connect(path) as conn:
        conn.executescript(SCHEMA)
        conn.executemany('INSERT INTO sources VALUES (?,?,?,?)',sources)
        conn.executemany(f'INSERT INTO entries ({",".join(columns)}) VALUES ({",".join("?" for _ in columns)})',entries)
