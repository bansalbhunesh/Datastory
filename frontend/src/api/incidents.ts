import { apiHeaders } from "./auth";

export type IncidentEntry = {
  id: string;
  createdAt: number;
  tableFQN: string;
  severity: "LOW" | "MEDIUM" | "HIGH";
  source: "deterministic" | "claude" | "llm";
};

export async function fetchIncidents(tableFQN: string, limit = 10): Promise<IncidentEntry[]> {
  const q = new URLSearchParams({ tableFQN, limit: String(limit) }).toString();
  const res = await fetch(`/api/incidents?${q}`, { headers: apiHeaders() });
  if (!res.ok) {
    return [];
  }
  const data = (await res.json()) as { incidents?: IncidentEntry[] };
  return data.incidents ?? [];
}
