import { Suspense, lazy, useCallback, useEffect, useRef, useState } from "react";
import { fetchReady, type ReadyResponse } from "./api/ready";
import { generateReport } from "./api/report";
import { fetchIncidents, type IncidentEntry } from "./api/incidents";
import { ExplainabilityPanel } from "./components/ExplainabilityPanel";
import { LoadingStages, type Stage } from "./components/LoadingStages";
import { ReportCard } from "./components/ReportCard";
import { SeverityBadge } from "./components/SeverityBadge";
import { ShareActions } from "./components/ShareActions";
import { StatusStrip } from "./components/StatusStrip";
import { TableQueryField } from "./components/TableQueryField";
import { sampleReport } from "./mock/sampleReport";
import type { GenerateReportResponse } from "./types/report";

// Heavy chart/graph deps (~250 KB minified each) lazy-load so the initial
// shell paints fast even on slow networks.
const LineageGraph = lazy(() =>
  import("./components/LineageGraph").then((m) => ({ default: m.LineageGraph })),
);
const IncidentChart = lazy(() =>
  import("./components/IncidentChart").then((m) => ({ default: m.IncidentChart })),
);

const EXAMPLE_QUERIES = ["dim_address", "fact_order", "stg_customers", "raw_shopify_orders"];

function downloadFilename(tableFQN: string) {
  const safe = tableFQN.replace(/[^a-zA-Z0-9_.-]/g, "_");
  const date = new Date().toISOString().split("T")[0];
  return `datastory-${safe}-${date}.md`;
}

export default function App() {
  const [mockMode, setMockMode] = useState(false);
  const [query, setQuery] = useState("dim_address");
  const [stage, setStage] = useState<Stage>("idle");
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<GenerateReportResponse | null>(null);
  const [incidents, setIncidents] = useState<IncidentEntry[]>([]);
  const [ready, setReady] = useState<ReadyResponse | null>(null);
  const [readyLoading, setReadyLoading] = useState(true);
  const [readyError, setReadyError] = useState<string | null>(null);

  // Holds active stage timers and any in-flight generate request id so we
  // can drop late results when the user submits again or unmounts.
  const stageTimersRef = useRef<number[]>([]);
  const generateSeqRef = useRef(0);

  const clearStageTimers = useCallback(() => {
    stageTimersRef.current.forEach((id) => window.clearTimeout(id));
    stageTimersRef.current = [];
  }, []);

  useEffect(() => () => clearStageTimers(), [clearStageTimers]);

  const loading = stage !== "idle" && stage !== "done";
  const severity = report?.severity ?? "UNKNOWN";

  useEffect(() => {
    let cancelled = false;
    setReadyLoading(true);
    setReadyError(null);
    fetchReady()
      .then((r) => { if (!cancelled) setReady(r); })
      .catch((e) => { if (!cancelled) setReadyError(e instanceof Error ? e.message : "ready failed"); })
      .finally(() => { if (!cancelled) setReadyLoading(false); });
    return () => { cancelled = true; };
  }, []);

  const onGenerate = useCallback(
    async (overrideQuery?: string) => {
      const q = (overrideQuery ?? query).trim();
      if (!q) return;

      setError(null);

      if (mockMode) {
        clearStageTimers();
        setReport(sampleReport);
        setIncidents([]);
        setStage("done");
        return;
      }

      // Bump sequence so any prior in-flight request becomes a no-op on resolve.
      const seq = ++generateSeqRef.current;
      clearStageTimers();
      setReport(null);
      setIncidents([]);
      setStage("resolving");

      stageTimersRef.current = [
        window.setTimeout(() => {
          if (generateSeqRef.current === seq) setStage("lineage");
        }, 800),
        window.setTimeout(() => {
          if (generateSeqRef.current === seq) setStage("quality");
        }, 2200),
        window.setTimeout(() => {
          if (generateSeqRef.current === seq) setStage("writing");
        }, 3800),
      ];

      try {
        const isFqn = q.includes(".");
        const res = await generateReport({
          query: isFqn ? "" : q,
          tableFQN: isFqn ? q : "",
        });
        if (generateSeqRef.current !== seq) return; // stale result
        clearStageTimers();
        setReport(res);
        setStage("done");
        try {
          const history = await fetchIncidents(res.tableFQN, 10);
          if (generateSeqRef.current === seq) setIncidents(history);
        } catch {
          // History is non-essential; keep the report even if it fails.
          if (generateSeqRef.current === seq) setIncidents([]);
        }
      } catch (e) {
        if (generateSeqRef.current !== seq) return;
        clearStageTimers();
        setStage("idle");
        setReport(null);
        setError(e instanceof Error ? e.message : "Unknown error");
      }
    },
    [query, mockMode, clearStageTimers]
  );

  return (
    <div className="min-h-full">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-10">

        {/* ── Header ── */}
        <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="text-xs font-semibold uppercase tracking-widest text-indigo-300">
              DataStory
            </div>
            <h1 className="mt-1 text-3xl font-semibold text-white">
              AI incident report generator
            </h1>
            <p className="mt-2 max-w-2xl text-sm text-slate-300">
              Paste a table name → get a full incident report with lineage graph,
              severity score, and remediation steps in seconds.
            </p>
          </div>
          <label
            htmlFor="mock-mode-toggle"
            className="flex cursor-pointer items-center gap-2 text-sm text-slate-300 hover:text-slate-100"
          >
            <input
              id="mock-mode-toggle"
              type="checkbox"
              checked={mockMode}
              onChange={(e) => setMockMode(e.target.checked)}
              className="h-4 w-4 accent-indigo-500"
            />
            Mock mode (offline demo)
          </label>
        </header>

        {/* ── Status ── */}
        <StatusStrip ready={ready} loading={readyLoading} error={readyError} />

        {/* ── Input ── */}
        <TableQueryField
          value={query}
          onChange={setQuery}
          onSubmit={() => onGenerate()}
          disabled={loading}
          mockMode={mockMode}
        />

        {/* ── Example queries (only when empty state) ── */}
        {!report && !loading && !error && (
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs text-slate-500">Try:</span>
            {EXAMPLE_QUERIES.map((q) => (
              <button
                key={q}
                type="button"
                onClick={() => {
                  setQuery(q);
                  onGenerate(q);
                }}
                className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-xs text-slate-300 transition-colors hover:border-indigo-500/60 hover:text-indigo-200"
              >
                {q}
              </button>
            ))}
          </div>
        )}

        {/* ── Loading stages ── */}
        <LoadingStages stage={stage} />

        {/* ── Error ── */}
        {error ? (
          <div
            className="flex items-start justify-between rounded-xl border border-rose-500/30 bg-rose-500/10 p-4 text-sm text-rose-100"
            role="alert"
          >
            <span>{error}</span>
            <button
              type="button"
              onClick={() => onGenerate()}
              className="ml-4 shrink-0 rounded-lg border border-rose-400/30 px-3 py-1 text-xs font-semibold transition-colors hover:border-rose-300 hover:text-white"
            >
              Retry
            </button>
          </div>
        ) : null}

        {/* ── Report ── */}
        {report ? (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">

            {/* Left: markdown report */}
            <div className="lg:col-span-2">
              <ReportCard
                title="Incident report"
                subtitle={report.tableFQN}
                markdown={report.markdown}
                source={report.source}
                warnings={report.warnings}
                filename={downloadFilename(report.tableFQN)}
              />
            </div>

            {/* Right: sidebar panels */}
            <div className="flex flex-col gap-4">
              <SeverityBadge severity={severity} />
              <ExplainabilityPanel report={report} />
              <Suspense fallback={<div className="rounded-xl border border-slate-800 bg-slate-900/40 p-4 text-xs text-slate-500">Loading lineage graph…</div>}>
                <LineageGraph lineage={report.lineage} />
              </Suspense>
              <Suspense fallback={null}>
                <IncidentChart incidents={incidents} />
              </Suspense>
              <ShareActions report={report} />

              {/* Failed tests */}
              <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-4">
                <div className="text-sm font-semibold text-slate-200">
                  Failed tests ({report.failedTests.length})
                </div>
                <ul className="mt-3 space-y-2 text-sm text-slate-300">
                  {report.failedTests.length === 0 ? (
                    <li className="text-slate-500">
                      None returned (or not configured in OpenMetadata).
                    </li>
                  ) : (
                    report.failedTests.map((t) => (
                      <li
                        key={t.name}
                        className="rounded-lg border border-slate-800 bg-slate-950/30 p-2"
                      >
                        <div className="font-medium text-slate-100">{t.name}</div>
                        <div className="text-xs text-rose-300/80">{t.status}</div>
                        {t.result ? (
                          <div className="mt-1 text-xs text-slate-400">{t.result}</div>
                        ) : null}
                      </li>
                    ))
                  )}
                </ul>
              </div>
            </div>
          </div>
        ) : (
          !loading && !error && (
            <div className="flex flex-col items-center justify-center rounded-xl border border-slate-800 bg-slate-900/30 px-6 py-14 text-center">
              <div className="text-3xl">📊</div>
              <div className="mt-3 text-sm font-medium text-slate-300">
                Enter a table name to generate an incident report
              </div>
              <div className="mt-1 text-xs text-slate-500">
                Powered by OpenMetadata lineage + Claude AI · works offline in mock mode
              </div>
            </div>
          )
        )}
      </div>
    </div>
  );
}
