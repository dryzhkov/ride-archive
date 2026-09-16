import copy
import io
import json
from pathlib import Path
import tempfile
import unittest
import zipfile
from ride_archive.query import DEFAULT, QueryError, compile_query, execute
from ride_archive.storage import seed, connect, export_bundle, restore_bundle

def q(*preds,entity='archive_entries',limit=100):
    return dict(version=1,entity=entity,where={'all':list(preds)},limit=limit)
def p(field,op,value=None):
    return dict(field=field,op=op,**({} if op=='is_null' else {'value':value}))

class Queries(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.path=Path(self.tmp.name)/'db.sqlite';seed(self.path)
        self.conn=connect(self.path)
    def tearDown(self):self.conn.close();self.tmp.cleanup()
    def ids(self,query):return {r['id'] for r in execute(self.conn,query)['records']}
    def test_corpus_queries(self):
        cases=[(q(),{'hells','kittitas','evans','tyee','alder'}),
               (q(entity='recordings'),{'hells','kittitas','evans'}),
               (q(p('trip_id','eq','touratech-rally-2025')),{'tyee','alder'}),
               (q(p('bike_id','eq','te-300'),entity='recordings'),{'evans','kittitas'}),
               (q(p('started_at','is_null')),{'tyee','alder'}),
               (q(p('raw_distance_m','gt',50*1609.344)),{'hells'}),
               (q(p('recorder','eq','Garmin watch'),p('exporter','eq','GaiaGPS')),{'kittitas'}),
               (q(p('has_quality_flags','eq',True)),{'hells'}),
               (q(p('bike_id','eq','te-300'),p('raw_distance_m','gt',50*1609.344)),set())]
        for query,want in cases:
            with self.subTest(query=query):self.assertEqual(self.ids(query),want)
    def test_unknown_not_zero(self):
        result=execute(self.conn,q(p('trip_id','eq','touratech-rally-2025'),p('raw_distance_m','gt',0)))
        self.assertEqual(result['total_count'],0);self.assertEqual(result['excluded_unknown_count'],4)
        # Two supplied routes lack measured distance; two other entries lack trip assignment.
        # A known false condition excludes; it is not counted as potentially unknown.
        result=execute(self.conn,q(p('bike_id','eq','te-300'),p('raw_distance_m','gt',0)))
        self.assertEqual(result['excluded_unknown_count'],0)
    def test_boundary_and_timezone(self):
        self.conn.execute('UPDATE entries SET raw_distance_m=? WHERE id=?',(80467.2,'evans'))
        self.assertNotIn('evans',self.ids(q(p('raw_distance_m','gt',80467.2))))
        self.assertIn('evans',self.ids(q(p('raw_distance_m','gte',80467.2))))
        self.assertEqual(self.ids(q(p('started_at','gte','2024-09-21T08:00:05-07:00'),p('started_at','lt','2024-09-21T15:00:06Z'))),{'evans'})
    def test_parameterization(self):
        self.assertEqual(self.ids(q(p('bike_id','eq',"' OR 1=1 --"))),set())
        self.assertEqual(len(self.ids(q())),5)
    def test_reject_invalid_contracts(self):
        bad=[q(p('difficulty','eq','hard')),q(p('bike_id','gt','a')),q(p('started_at','eq','2025-01-01')),
             q(p('raw_distance_m','gt',float('nan'))),q(p('raw_distance_m','gt',True)),
             q(p('bike_id','in',[])),q(limit=101),q(limit=True),q(p('has_quality_flags','eq',1)),
             {**q(),'sql':'DELETE FROM entries'},q(*[p('kind','eq','recorded')]*21),
             {**q(),'where':{'any':[]}}, {**q(),'order_by':[{'field':'id; DROP TABLE entries','direction':'asc'}]}]
        for query in bad:
            with self.subTest(query=query):
                with self.assertRaises(QueryError):compile_query(query)
    def test_limit_reports_total(self):
        result=execute(self.conn,q(limit=2));self.assertEqual(result['total_count'],5)
        self.assertEqual(len(result['records']),2);self.assertTrue(result['truncated'])
    def test_seed_idempotent(self):
        seed(self.path);self.assertEqual(self.conn.execute('SELECT count(*) FROM sources').fetchone()[0],5)
    def test_round_trip(self):
        bundle=export_bundle(self.conn);path=Path(self.tmp.name)/'restored.sqlite';restore_bundle(bundle,path)
        with connect(path,True) as restored:
            self.assertEqual(execute(self.conn,q()),execute(restored,q()))
            self.assertEqual([tuple(r) for r in self.conn.execute('SELECT sha256,original FROM sources ORDER BY id')],
                             [tuple(r) for r in restored.execute('SELECT sha256,original FROM sources ORDER BY id')])
        with self.assertRaises(ValueError):restore_bundle(bundle,path)
    def test_partial_export_and_checksum(self):
        bundle=export_bundle(self.conn,{'tyee','alder'},q(p('trip_id','eq','touratech-rally-2025')))
        path=Path(self.tmp.name)/'partial.sqlite';restore_bundle(bundle,path)
        with connect(path,True) as restored:self.assertEqual(execute(restored,q())['total_count'],2)
        stream=io.BytesIO()
        with zipfile.ZipFile(io.BytesIO(bundle)) as src,zipfile.ZipFile(stream,'w') as dst:
            for member in src.namelist():dst.writestr(member,b'changed' if member.endswith('.gpx') else src.read(member))
        with self.assertRaises(ValueError):restore_bundle(stream.getvalue(),Path(self.tmp.name)/'bad.sqlite')
    def test_query_connection_readonly(self):
        with connect(self.path,True) as conn:
            with self.assertRaises(Exception):conn.execute('DELETE FROM entries')

if __name__=='__main__':unittest.main()
