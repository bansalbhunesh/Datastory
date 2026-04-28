import type { GenerateReportResponse, LineageSummary } from "../types/report";
import { apiHeaders } from "./auth";

export type GenerateReportRequest = {
  query?: string;
  tableFQN?: string;
};

const VALID_SEVERITIES = ["LOW", "MEDIUM", "HIGH", "UNKNOWN"] as const;
type Severity = (typeof VALID_SEVERITIES)[number];

function isObject(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null;
}

function asStringArray(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : [];
}

function asLineage(v: unknown): LineageSummary {
  if (!isObject(v)) {
    return { focal: "", upstream: [], downstream: [], upstreamRaw: 0, downstreamRaw: 0 };
  }
  return {
    focal: typeof v.focal === "string" ? v.focal : "",
    upstream: asStringArray(v.upstream),
    downstream: asStringArray(v.downstream),
    upstreamRaw: typeof v.upstreamRaw === "number" ? v.upstreamRaw : 0,
    downstreamRaw: typeof v.downstreamRaw === "number" ? v.downstreamRaw : 0,
  };
}

function normalizeReport(raw: unknown): GenerateReportResponse {
  if (!isObject(raw)) {
    throw new Error("Server returned an unexpected response shape.");
  }
  const tableFQN = typeof raw.tableFQN === "string" ? raw.tableFQN : "";
  const markdown = typeof raw.markdown === "string" ? raw.markdown : "";
  if (!tableFQN || !markdown) {
    throw new Error("Server response missing required fields (tableFQN, markdown).");
  }
  const sevRaw = typeof raw.severity === "string" ? raw.severity.toUpperCase() : "UNKNOWN";
  const severity: Severity =
    (VALID_SEVERITIES as readonly string[]).includes(sevRaw) ? (sevRaw as Severity) : "UNKNOWN";
  const sourceRaw = typeof raw.source === "string" ? raw.source : "";
  const source: "claude" | "deterministic" =
    sourceRaw === "claude" || sourceRaw === "llm" ? "claude" : "deterministic";

  const failedTestsArr = Array.isArray(raw.failedTests) ? raw.failedTests : [];
  const failedTests = failedTestsArr
    .filter(isObject)
    .map((t) => ({
      name: typeof t.name === "string" ? t.name : "",
      fqn: typeof t.fqn === "string" ? t.fqn : undefined,
      status: typeof t.status === "string" ? t.status : "",
      result: typeof t.result === "string" ? t.result : undefined,
      description: typeof t.description === "string" ? t.description : undefined,
      updatedAt: typeof t.updatedAt === "number" ? t.updatedAt : undefined,
    }))
    .filter((t) => t.name);

  const expRaw = isObject(raw.explanation) ? raw.explanation : undefined;
  const explanation = expRaw
    ? {
        failedTestCount: typeof expRaw.failedTestCount === "number" ? expRaw.failedTestCount : 0,
        downstreamCount: typeof expRaw.downstreamCount === "number" ? expRaw.downstreamCount : 0,
        upstreamCount: typeof expRaw.upstreamCount === "number" ? expRaw.upstreamCount : 0,
        lineageComplete: Boolean(expRaw.lineageComplete),
        confidence: typeof expRaw.confidence === "number" ? expRaw.confidence : 0,
      }
    : undefined;

  return {
    tableFQN,
    markdown,
    severity,
    summary: typeof raw.summary === "string" ? raw.summary : undefined,
    rootCauses: asStringArray(raw.rootCauses),
    impacts: asStringArray(raw.impacts),
    remediation: asStringArray(raw.remediation),
    lineage: asLineage(raw.lineage),
    failedTests,
    explanation,
    source,
    warnings: asStringArray(raw.warnings),
  };
}

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

  const raw: unknown = await res.json();
  return normalizeReport(raw);
}

// Exported for tests.
export const __test = { normalizeReport };
