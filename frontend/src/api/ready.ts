import { apiHeaders } from "./auth";

export type ReadyResponse = {
  openmetadata: { reachable: boolean; auth: boolean };
  claude: { configured: boolean };
};

export async function fetchReady(): Promise<ReadyResponse> {
  const res = await fetch("/api/ready", { headers: apiHeaders() });
  if (!res.ok) {
    throw new Error(`ready: HTTP ${res.status}`);
  }
  return (await res.json()) as ReadyResponse;
}
