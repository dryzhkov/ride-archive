export interface CatalogItem { id: string; name: string; owner_id: string; created_at: string }
export interface Entry {
  id: string; owner_id: string; title: string;
  kind: 'unclassified' | 'recorded' | 'route_reference';
  trip_id: string | null; bike_id: string | null;
  revision: number; created_at: string; updated_at: string;
}
export interface Source { original_filename: string; id: string; sha256: string; byte_length: number; created_at: string }
export interface Page<T> { items: T[]; next_offset: number | null }
export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message); }
}
let token = '';
// The personal API token lives only in memory, never localStorage or a URL.
export function setToken(value: string) { token = value; }
export async function request(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  if (token) headers.set('Authorization', `Bearer ${token}`);
  const response = await fetch(`/api/v1${path}`, { ...init, headers });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(response.status, body?.error?.message ?? `Request failed (${response.status}).`);
  }
  return response;
}
export async function api<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  return (await request(path, { method, ...(body === undefined ? {} : {
    headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
  }) })).json();
}
export async function allCatalog(kind: 'trips' | 'bikes'): Promise<CatalogItem[]> {
  const items: CatalogItem[] = []; let offset: number | null = 0;
  while (offset !== null) {
    const page: Page<CatalogItem> = await api(`/${kind}?limit=100&offset=${offset}`);
    items.push(...page.items); offset = page.next_offset;
  }
  return items;
}

export interface PathSummary {
  id: string; source_id: string; original_filename: string; name: string;
  structure: 'gpx_track' | 'gpx_route'; source_locator: string;
  point_count: number; segment_count: number;
}
export interface TrackPoint { latitude: number; longitude: number; elevation_m: number | null; raw_time?: string }
export interface PathGeometry extends PathSummary { segments: TrackPoint[][] }
export async function allRecords<T>(kind: 'entries' | 'sources'): Promise<T[]> {
  const items: T[] = []; let offset: number | null = 0;
  while (offset !== null) {
    const page: Page<T> = await api(`/${kind}?limit=100&offset=${offset}`);
    items.push(...page.items); offset = page.next_offset;
  }
  return items;
}
