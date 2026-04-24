export type LineageSummary = {
  focal: string;
  upstream: string[];
  downstream: string[];
  upstreamRaw: number;
  downstreamRaw: number;
};

export type FailedTestSummary = {
  name: string;
  fqn?: string;
  status: string;
  result?: string;
  updatedAt?: number;
  description?: string;
};

export type SeverityExplanation = {
  failedTestCount: number;
  downstreamCount: number;
  upstreamCount: number;
  lineageComplete: boolean;
  confidence: number;
};

export type GenerateReportResponse = {
  tableFQN: string;
  markdown: string;
  severity: "LOW" | "MEDIUM" | "HIGH" | "UNKNOWN";
  summary?: string;
  rootCauses?: string[];
  impacts?: string[];
  remediation?: string[];
  lineage: LineageSummary;
  failedTests: FailedTestSummary[];
  explanation?: SeverityExplanation;
  source: "claude" | "llm" | "deterministic";
  warnings?: string[];
};
