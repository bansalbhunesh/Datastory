import type { ReadyResponse } from "../api/ready";

export function StatusStrip({
  ready,
  loading,
  error,
}: {
  ready: ReadyResponse | null;
  loading: boolean;
  error: string | null;
}) {
  if (loading) {
    return (
      <div className="rounded-xl border border-slate-800 bg-slate-900/40 px-4 py-3 text-sm text-slate-400">
        Checking backend / OpenMetadata readiness…
      </div>
    );
  }
  if (error) {
    return (
      <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
        Readiness check failed: {error}
      </div>
    );
  }
  if (!ready) return null;

  const omOk = ready.openmetadata.reachable && ready.openmetadata.auth;
  const claudeOk = ready.claude.configured;

  return (
    <div className="flex flex-col gap-2 rounded-xl border border-slate-800 bg-slate-900/40 px-4 py-3 text-sm text-slate-200 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex flex-wrap gap-3">
        <span className={omOk ? "text-emerald-200" : "text-amber-200"}>
          OpenMetadata: {ready.openmetadata.reachable ? (ready.openmetadata.auth ? "reachable + auth" : "reachable, needs token/login") : "not reachable"}
        </span>
        <span className={claudeOk ? "text-emerald-200" : "text-slate-400"}>
          Claude: {claudeOk ? "API key set" : "not configured (deterministic draft)"}
        </span>
      </div>
      {!omOk ? <span className="text-xs text-slate-500">Tip: run `make mock`, then refresh.</span> : null}
    </div>
  );
}
