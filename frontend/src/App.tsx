import { useEffect, useMemo, useState } from "react";
import { fetchReady, type ReadyResponse } from "./api/ready";
import { generateReport } from "./api/report";
import { fetchIncidents, type IncidentEntry } from "./api/incidents";
import { LineageTrace } from "./components/LineageTrace";
import { ReportCard } from "./components/ReportCard";
import { SeverityBadge } from "./components/SeverityBadge";
import { StatusStrip } from "./components/StatusStrip";
import { TableQueryField } from "./components/TableQueryField";
import { sampleReport } from "./mock/sampleReport";
import type { GenerateReportResponse } from "./types/report";

function guessSeverity(r: GenerateReportResponse): "LOW" | "MEDIUM" | "HIGH" | "UNKNOWN" {
  const fails = r.failedTests.length;
  const down = r.lineage.downstream.length;
  if (fails >= 2 || (fails >= 1 && down >= 3)) return "HIGH";
  if (fails === 1 || down >= 2) return "MEDIUM";
  if (fails === 0 && down <= 1) return "LOW";
  return "UNKNOWN";
}

export default function App() {
  const [mockMode, setMockMode] = useState(false);
  const [query, setQuery] = useState("dim_address");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<GenerateReportResponse | null>(null);
  const [incidents, setIncidents] = useState<IncidentEntry[]>([]);

  const [ready, setReady] = useState<ReadyResponse | null>(null);
  const [readyLoading, setReadyLoading] = useState(true);
  const [readyError, setReadyError] = useState<string | null>(null);

  const severity = useMemo(() => {
    if (!report) return "UNKNOWN";
    return report.severity ?? guessSeverity(report);
  }, [report]);

  useEffect(() => {
    let cancelled = false;
    setReadyLoading(true);
    setReadyError(null);
    fetchReady()
      .then((r) => {
        if (!cancelled) setReady(r);
      })
      .catch((e) => {
        if (!cancelled) setReadyError(e instanceof Error ? e.message : "ready failed");
      })
      .finally(() => {
        if (!cancelled) setReadyLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function onGenerate() {
    setError(null);
    if (mockMode) {
      setReport(sampleReport);
      return;
    }

    setLoading(true);
    try {
      const q = query.trim();
      const isFqn = q.includes(".");
      const res = await generateReport({
        query: isFqn ? "" : q,
        tableFQN: isFqn ? q : "",
      });
      setReport(res);
      const history = await fetchIncidents(res.tableFQN, 8);
      setIncidents(history);
    } catch (e) {
      setReport(null);
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-full">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-10">
        <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="text-xs font-semibold uppercase tracking-wide text-indigo-300">DataStory</div>
            <h1 className="mt-1 text-3xl font-semibold text-white">AI incident report generator</h1>
            <p className="mt-2 max-w-2xl text-sm text-slate-300">
              OpenMetadata supplies lineage and failing tests; the API turns that into a postmortem-style Markdown report (Claude when
              configured, otherwise a deterministic draft).
            </p>
          </div>
          <label htmlFor="mock-mode-toggle" className="flex items-center gap-2 text-sm text-slate-200">
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

        <StatusStrip ready={ready} loading={readyLoading} error={readyError} />

        <TableQueryField
          value={query}
          onChange={setQuery}
          onSubmit={onGenerate}
          disabled={loading}
          mockMode={mockMode}
        />

        {loading ? (
          <div className="rounded-xl border border-slate-800 bg-slate-900/30 px-4 py-6 text-sm text-slate-400" aria-live="polite">
            Generating report…
          </div>
        ) : null}

        {error ? (
          <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 p-4 text-sm text-rose-100" role="alert">
            {error}
          </div>
        ) : null}

        {report ? (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <div className="lg:col-span-2">
              <ReportCard
                title="Incident report"
                subtitle={report.tableFQN}
                markdown={report.markdown}
                source={report.source === "llm" ? "claude" : report.source}
                warnings={report.warnings}
              />
            </div>
            <div className="flex flex-col gap-4">
              <div className="flex flex-wrap items-center gap-2">
                <SeverityBadge severity={severity} />
              </div>
              <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-4 text-xs text-slate-300">
                <div className="font-semibold text-slate-100">Why this severity?</div>
                <div className="mt-1">
                  Failed tests: <span className="font-semibold">{report.failedTests.length}</span> | Downstream tables:{" "}
                  <span className="font-semibold">{report.lineage.downstream.length}</span>
                </div>
              </div>
              <LineageTrace lineage={report.lineage} />
              <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-4">
                <div className="text-sm font-semibold text-slate-200">Recent incidents</div>
                <ul className="mt-3 space-y-2 text-xs text-slate-300">
                  {incidents.length === 0 ? (
                    <li className="text-slate-500">No saved incidents yet for this table.</li>
                  ) : (
                    incidents.map((i) => (
                      <li key={i.id} className="rounded-lg border border-slate-800 bg-slate-950/30 p-2">
                        <div className="font-medium text-slate-100">{new Date(i.createdAt * 1000).toLocaleString()}</div>
                        <div className="text-slate-400">
                          {i.severity} via {i.source === "llm" ? "claude" : i.source}
                        </div>
                      </li>
                    ))
                  )}
                </ul>
              </div>
              <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-4">
                <div className="text-sm font-semibold text-slate-200">Failed tests ({report.failedTests.length})</div>
                <ul className="mt-3 space-y-2 text-sm text-slate-300">
                  {report.failedTests.length === 0 ? (
                    <li className="text-slate-500">None returned (or not configured).</li>
                  ) : (
                    report.failedTests.map((t) => (
                      <li key={t.name} className="rounded-lg border border-slate-800 bg-slate-950/30 p-2">
                        <div className="font-medium text-slate-100">{t.name}</div>
                        <div className="text-xs text-slate-400">{t.status}</div>
                        {t.result ? <div className="mt-1 text-xs text-slate-300">{t.result}</div> : null}
                      </li>
                    ))
                  )}
                </ul>
              </div>
            </div>
          </div>
        ) : (
          !loading && (
            <div className="rounded-xl border border-slate-800 bg-slate-900/30 p-6 text-sm text-slate-400">
              Generate a report to see lineage, failures, and the incident narrative. Toggle mock mode for a guaranteed demo without Docker.
            </div>
          )
        )}
      </div>
    </div>
  );
}
