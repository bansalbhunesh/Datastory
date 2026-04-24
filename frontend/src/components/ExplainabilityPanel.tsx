import type { GenerateReportResponse } from "../types/report";

type Props = { report: GenerateReportResponse };

type Row = {
  label: string;
  value: string;
  status: "bad" | "warn" | "good" | "neutral";
  weight: string;
};

export function ExplainabilityPanel({ report }: Props) {
  const { explanation, severity } = report;
  if (!explanation) return null;

  const rows: Row[] = [
    {
      label: "Failed quality tests",
      value: String(explanation.failedTestCount),
      status:
        explanation.failedTestCount >= 2
          ? "bad"
          : explanation.failedTestCount === 1
          ? "warn"
          : "good",
      weight: "Primary signal",
    },
    {
      label: "Downstream tables at risk",
      value: String(explanation.downstreamCount),
      status:
        explanation.downstreamCount >= 3
          ? "bad"
          : explanation.downstreamCount > 0
          ? "warn"
          : "good",
      weight: "Impact scope",
    },
    {
      label: "Upstream dependencies",
      value: String(explanation.upstreamCount),
      status: "neutral",
      weight: "Context",
    },
    {
      label: "Lineage data",
      value: explanation.lineageComplete ? "Available" : "Unavailable",
      status: explanation.lineageComplete ? "good" : "warn",
      weight: "Accuracy factor",
    },
  ];

  const dotColor: Record<Row["status"], string> = {
    bad: "bg-rose-400",
    warn: "bg-amber-400",
    good: "bg-emerald-400",
    neutral: "bg-slate-500",
  };
  const textColor: Record<Row["status"], string> = {
    bad: "text-rose-300",
    warn: "text-amber-300",
    good: "text-emerald-300",
    neutral: "text-slate-300",
  };

  const confColor =
    explanation.confidence >= 80
      ? "bg-emerald-500"
      : explanation.confidence >= 55
      ? "bg-amber-500"
      : "bg-rose-500";

  const sevColor: Record<string, string> = {
    HIGH: "text-rose-300",
    MEDIUM: "text-amber-300",
    LOW: "text-emerald-300",
    UNKNOWN: "text-slate-400",
  };

  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-4">
      <div className="flex items-center justify-between">
        <div className="text-sm font-semibold text-slate-200">Severity breakdown</div>
        <div className={`text-xs font-bold ${sevColor[severity ?? "UNKNOWN"]}`}>
          {severity}
        </div>
      </div>

      <div className="mt-3 space-y-2.5">
        {rows.map((row) => (
          <div key={row.label} className="flex items-center justify-between text-xs">
            <div className="flex items-center gap-2">
              <div
                className={`h-1.5 w-1.5 flex-shrink-0 rounded-full ${dotColor[row.status]}`}
              />
              <span className="text-slate-400">{row.label}</span>
            </div>
            <div className="flex items-center gap-3">
              <span className="text-[10px] text-slate-600">{row.weight}</span>
              <span className={`font-semibold ${textColor[row.status]}`}>
                {row.value}
              </span>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-4">
        <div className="mb-1.5 flex items-center justify-between text-xs">
          <span className="text-slate-400">Confidence score</span>
          <span className="font-semibold text-slate-200">{explanation.confidence}%</span>
        </div>
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-800">
          <div
            className={`h-1.5 rounded-full transition-all duration-700 ${confColor}`}
            style={{ width: `${explanation.confidence}%` }}
          />
        </div>
        <div className="mt-1 text-[10px] text-slate-500">
          Based on test failures, lineage completeness, and downstream scope
        </div>
      </div>
    </div>
  );
}
