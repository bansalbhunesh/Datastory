import { apiHeaders } from "./auth";

export type TableSearchHit = {
  id: string;
  name: string;
  fullyQualifiedName: string;
};

export async function searchTables(q: string, signal?: AbortSignal): Promise<TableSearchHit[]> {
  const params = new URLSearchParams({ q });
  const res = await fetch(`/api/search/tables?${params.toString()}`, {
    headers: apiHeaders(),
    signal,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  const body = (await res.json()) as { hits?: TableSearchHit[] };
  return Array.isArray(body.hits) ? body.hits : [];
}
