"""Loopback-only PoC API + existing map UI. Python standard library only."""
import argparse
import datetime as dt
import json
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse
from .query import QueryError, execute, FIELDS
from .storage import ROOT, DB, connect, seed, export_bundle, restore_bundle

CATALOG={'translation':{'enabled':False,'reason':'AI is not configured. Use examples or edit the structured query.'},
         'fields':FIELDS,'bikes':{'ktm-890':['890','KTM 890 Adventure R'],'te-300':['TE 300','Husqvarna TE 300']},
         'trips':{'touratech-rally-2025':'Touratech Rally 2025','hells-canyon-2026':'Hells Canyon 2026'},
         'limits':{'predicates':20,'results':100}}

class Handler(SimpleHTTPRequestHandler):
    def __init__(self,*args,**kwargs):
        super().__init__(*args,directory=str(ROOT/'prototype'),**kwargs)

    def log_message(self,format,*args):
        # Do not log query text, coordinates or request bodies.
        pass

    def trusted(self):
        port=self.server.server_port
        hosts={f'127.0.0.1:{port}',f'localhost:{port}'}
        if self.headers.get('Host') not in hosts: return False
        origin=self.headers.get('Origin')
        return origin is None or origin in {f'http://{h}' for h in hosts}

    def reply(self,data,status=200,kind='application/json',filename=None):
        body=data if isinstance(data,bytes) else json.dumps(data,allow_nan=False).encode()
        self.send_response(status);self.send_header('Content-Type',kind);self.send_header('Content-Length',str(len(body)))
        self.send_header('Cache-Control','no-store');self.send_header('X-Content-Type-Options','nosniff')
        if filename:self.send_header('Content-Disposition',f'attachment; filename="{filename}"')
        self.end_headers();self.wfile.write(body)

    def do_GET(self):
        if not self.trusted(): return self.reply({'error':'Untrusted request origin/host.'},403)
        path=urlparse(self.path).path
        if path=='/api/catalog': return self.reply(CATALOG)
        if path=='/api/export':
            with connect(self.server.db,True) as conn: body=export_bundle(conn)
            return self.reply(body,kind='application/zip',filename='ride-archive.zip')
        if path.startswith('/api/'):return self.reply({'error':'Unknown endpoint.'},404)
        # Serve only the small UI asset set; never expose project files or DB.
        if path not in ('/','/index.html','/app.js','/query-ui.js','/styles.css'):
            return self.reply({'error':'Not found.'},404)
        super().do_GET()

    def do_HEAD(self):
        if not self.trusted():return self.reply({'error':'Forbidden'},403)
        if urlparse(self.path).path not in ('/','/index.html','/app.js','/query-ui.js','/styles.css'):return self.reply({'error':'Not found'},404)
        super().do_HEAD()

    def do_POST(self):
        if not self.trusted(): return self.reply({'error':'Untrusted request origin/host.'},403)
        if self.headers.get('Content-Type','').split(';')[0]!='application/json':return self.reply({'error':'Use application/json.'},415)
        try:
            length=int(self.headers.get('Content-Length','0'))
            if not 0<length<=32768:raise QueryError('Request must be 1–32768 bytes.')
            query=json.loads(self.rfile.read(length),parse_constant=lambda s: (_ for _ in ()).throw(QueryError('Non-finite JSON number.')))
            path=urlparse(self.path).path
            if path not in ('/api/query','/api/export-results'):return self.reply({'error':'Unknown endpoint.'},404)
            with connect(self.server.db,True) as conn:
                result=execute(conn,query)
                if path=='/api/export-results':
                    body=export_bundle(conn,{r['id'] for r in result['records']},result['query'])
                    return self.reply(body,kind='application/zip',filename='ride-archive-results.zip')
            return self.reply(result)
        except (QueryError,ValueError,TypeError,KeyError) as exc:
            return self.reply({'error':str(exc)},400)
        except Exception:
            return self.reply({'error':'Internal query failure.'},500)

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--port',type=int,default=8766);parser.add_argument('--db',type=Path,default=DB)
    parser.add_argument('--init-only',action='store_true');parser.add_argument('--restore',type=Path)
    args=parser.parse_args()
    if args.restore:
        restore_bundle(args.restore.read_bytes(),args.db)
    elif not args.db.exists():seed(args.db)
    if args.init_only:return
    server=ThreadingHTTPServer(('127.0.0.1',args.port),Handler);server.db=args.db
    print(f'Ride Archive: http://127.0.0.1:{args.port}',flush=True)
    server.serve_forever()

if __name__=='__main__':main()
