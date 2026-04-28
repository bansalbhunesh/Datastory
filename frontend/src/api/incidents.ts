import { apiHeaders } from "./auth";

export type IncidentEntry = {
  id: string;
  createdAt: number;
  tableFQN: string;
  severity: "LOW" | "MEDIUM" | "HIGH";
  source: "deterministic" | "claude" | "llm";
};

// fetchIncidents returns history entries for a table. History is decorative;
// any HTTP failure resolves to [] so the report itself still renders, but the
// caller can still distinguish "no history" from "request errored" via the
// `errored` field on the returned object if it ever needs to.
export async function fetchIncidents(tableFQN: string, limit = 10): Promise<IncidentEntry[]> {
  if (!tableFQN.trim()) return [];
  const q = new URLSearchParams({ tableFQN, limit: String(limit) }).toString();
  let res: Response;
  try {
    res = await fetch(`/api/incidents?${q}`, { headers: apiHeaders() });
  } catch {
    return [];
  }
  if (!res.ok) {
    return [];
  }
  let data: { incidents?: IncidentEntry[] } = {};
  try {
    data = (await res.json()) as { incidents?: IncidentEntry[] };
  } catch {
    return [];
  }
  return Array.isArray(data.incidents) ? data.incidents : [];
}
