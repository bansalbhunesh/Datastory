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

export type GenerateReportResponse = {
  tableFQN: string;
  markdown: string;
  lineage: LineageSummary;
  failedTests: FailedTestSummary[];
};
