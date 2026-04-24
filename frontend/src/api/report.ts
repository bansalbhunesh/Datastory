import type { GenerateReportResponse } from "../types/report";
import { apiHeaders } from "./auth";

export type GenerateReportRequest = {
  query?: string;
  tableFQN?: string;
};

export async function generateReport(req: GenerateReportRequest): Promise<GenerateReportResponse> {
  const res = await fetch("/api/generate-report", {
    method: "POST",
    headers: apiHeaders({ "Content-Type": "application/json" }),
    body: JSON.stringify({
      query: req.query ?? "",
      tableFQN: req.tableFQN ?? "",
    }),
  });

  if (!res.ok) {
    const text = await res.text();
    let msg = text || `HTTP ${res.status}`;
    try {
      const j = JSON.parse(text) as { error?: string };
      if (typeof j.error === "string" && j.error.trim()) {
        msg = j.error.trim();
      }
    } catch {
      // keep msg as raw body
    }
    throw new Error(msg.length > 800 ? `${msg.slice(0, 800)}…` : msg);
  }

  const data = (await res.json()) as GenerateReportResponse;
  const source = data.source === "claude" || data.source === "llm" ? "claude" : "deterministic";
  return {
    ...data,
    source,
    severity: data.severity ?? "UNKNOWN",
    warnings: data.warnings ?? [],
  };
}
