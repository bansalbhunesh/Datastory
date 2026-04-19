import type { GenerateReportResponse } from "../types/report";

export type GenerateReportRequest = {
  query?: string;
  tableFQN?: string;
};

export async function generateReport(req: GenerateReportRequest): Promise<GenerateReportResponse> {
  const res = await fetch("/api/generate-report", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      query: req.query ?? "",
      tableFQN: req.tableFQN ?? "",
    }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }

  return (await res.json()) as GenerateReportResponse;
}
