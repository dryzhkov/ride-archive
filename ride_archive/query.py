"""Versioned, read-only query compiler. No caller-controlled SQL identifiers."""
import copy
import datetime as dt
import math

FIELDS = {
    'id': 'text', 'kind': 'text', 'bike_id': 'text', 'trip_id': 'text',
    'recorder': 'text', 'exporter': 'text', 'started_at': 'time',
    'raw_distance_m': 'number', 'route_length_m': 'number',
    'user_reported_year': 'integer', 'user_reports_rode_route': 'boolean',
    'has_quality_flags': 'boolean',
}
OPS = {'eq': '=', 'gt': '>', 'gte': '>=', 'lt': '<', 'lte': '<='}
DEFAULT = {'version': 1, 'entity': 'archive_entries', 'where': {'all': []},
           'order_by': [{'field': 'started_at', 'direction': 'desc'}], 'limit': 100}

class QueryError(ValueError):
    pass

def keys(obj, required, optional=()):
    if not isinstance(obj, dict) or set(obj)-set(required)-set(optional) or set(required)-set(obj):
        raise QueryError('Missing or unsupported object fields.')

def value_for(field, value):
    kind = FIELDS[field]
    if kind == 'boolean':
        if type(value) is not bool: raise QueryError(f'{field} requires a boolean.')
    elif kind in ('number', 'integer'):
        if type(value) not in (int, float) or (type(value) is int and not -(2**63) <= value < 2**63) or (type(value) is float and not math.isfinite(value)): raise QueryError(f'{field} requires a finite number.')
        if kind == 'integer' and type(value) is not int: raise QueryError(f'{field} requires an integer.')
        if value < 0: raise QueryError(f'{field} cannot be negative.')
    elif not isinstance(value, str) or not value or len(value) > 200:
        raise QueryError(f'{field} requires a nonempty string (max 200 characters).')
    if kind == 'time':
        try:
            instant = dt.datetime.fromisoformat(value.replace('Z', '+00:00'))
            if instant.tzinfo is None: raise ValueError()
            value = instant.astimezone(dt.timezone.utc).isoformat(timespec='microseconds')
        except (ValueError, OverflowError): raise QueryError('Date values require an ISO timestamp with timezone.')
    if field == 'kind' and value not in ('recorded', 'route_reference', 'unknown'):
        raise QueryError('Invalid entry kind.')
    return value

def compile_query(query):
    keys(query, ('version', 'entity', 'where'), ('order_by', 'limit'))
    if type(query['version']) is not int or query['version'] != 1: raise QueryError('Only query version 1 is supported.')
    if query['entity'] not in ('archive_entries', 'recordings'): raise QueryError('Unsupported entity.')
    keys(query['where'], ('all',))
    predicates = query['where']['all']
    if not isinstance(predicates, list) or len(predicates) > 20: raise QueryError('Use at most 20 AND predicates.')
    limit = query.get('limit', 100)
    if type(limit) is not int or not 1 <= limit <= 100: raise QueryError('Limit must be an integer from 1 to 100.')
    order = query.get('order_by', DEFAULT['order_by'])
    if not isinstance(order, list) or not 1 <= len(order) <= 3: raise QueryError('Use one to three ordering fields.')
    ordering = []
    for item in order:
        keys(item, ('field', 'direction'))
        field, direction = item['field'], item['direction']
        if not isinstance(field,str) or field not in FIELDS or direction not in ('asc','desc'): raise QueryError('Invalid ordering.')
        ordering.extend([f'{field} IS NULL ASC', f'{field} {direction.upper()}'])
    ordering.append('id ASC')
    scope = "kind = 'recorded'" if query['entity'] == 'recordings' else '1=1'
    clauses, params, normalized = [], [], []
    for pred in predicates:
        keys(pred, ('field','op'), ('value',))
        field, op = pred['field'], pred['op']
        if not isinstance(field,str) or field not in FIELDS: raise QueryError('Unsupported query field.')
        if not isinstance(op,str): raise QueryError('Unsupported operator.')
        if op == 'is_null':
            if 'value' in pred: raise QueryError('is_null takes no value.')
            clauses.append(f'{field} IS NULL'); normalized.append(dict(pred)); continue
        if 'value' not in pred: raise QueryError('Predicate value is required.')
        if op == 'in':
            if not isinstance(pred['value'],list) or not 1 <= len(pred['value']) <= 20: raise QueryError('in requires 1–20 values.')
            values = [value_for(field,v) for v in pred['value']]
            clauses.append(f'{field} IN ({",".join("?" for _ in values)})');params.extend(values)
        else:
            if op not in OPS or (op != 'eq' and FIELDS[field] not in ('number','integer','time')): raise QueryError('Operator is not supported for this field.')
            values = value_for(field,pred['value'])
            clauses.append(f'{field} {OPS[op]} ?');params.append(values)
        normalized.append({'field':field,'op':op,'value':values})
    expression = ' AND '.join(f'({c})' for c in clauses) or '1=1'
    canonical = dict(version=1,entity=query['entity'],where={'all':normalized},order_by=copy.deepcopy(order),limit=limit)
    return canonical, scope, expression, params, ', '.join(ordering)

def execute(conn, query):
    canonical, scope, expr, params, ordering = compile_query(query)
    # Three-valued SQL logic: FALSE excludes; NULL means the result is unknown.
    total = conn.execute(f'SELECT count(*) FROM entries WHERE ({scope}) AND ({expr})', params).fetchone()[0]
    unknown = conn.execute(f'SELECT count(*) FROM entries WHERE ({scope}) AND (({expr}) IS NULL)', params).fetchone()[0]
    rows = conn.execute(f'SELECT * FROM entries WHERE ({scope}) AND ({expr}) ORDER BY {ordering} LIMIT ?', [*params, canonical['limit']]).fetchall()
    import json
    records=[]
    for row in rows:
        fields=dict(row)
        for field,kind in FIELDS.items():
            if kind=='boolean' and fields[field] is not None: fields[field]=bool(fields[field])
        records.append({'id':row['id'], 'fields':{f:fields[f] for f in FIELDS},
                        'matched_fields':{p['field']:fields[p['field']] for p in canonical['where']['all']},
                        'provenance':json.loads(row['provenance']), 'display':json.loads(row['display'])})
    readable = [f"{p['field']} {p['op']} {json.dumps(p.get('value',''),ensure_ascii=False)}".strip() for p in canonical['where']['all']]
    return dict(query=canonical,interpretation=('GPS recordings' if canonical['entity']=='recordings' else 'All archive entry types')+(' · '+' AND '.join(readable) if readable else ''), records=records,total_count=total,excluded_unknown_count=unknown,truncated=total>len(records))
