import type { LineageSummary } from "../types/report";

export function LineageTrace({ lineage }: { lineage: LineageSummary }) {
  const rows: { label: string; items: string[] }[] = [
    { label: "Upstream", items: lineage.upstream },
    { label: "Focal table", items: lineage.focal ? [lineage.focal] : [] },
    { label: "Downstream", items: lineage.downstream },
  ];

  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-4">
      <div className="text-sm font-semibold text-slate-200">Lineage trace (text)</div>
      <div className="mt-3 space-y-3 text-sm text-slate-300">
        {rows.map((r) => (
          <div key={r.label}>
            <div className="text-xs font-semibold uppercase tracking-wide text-slate-400">{r.label}</div>
            {r.items.length === 0 ? (
              <div className="mt-1 text-slate-500">—</div>
            ) : (
              <ol className="mt-1 list-decimal space-y-1 pl-5">
                {r.items.map((x) => (
                  <li key={x} className="break-all">
                    {x}
                  </li>
                ))}
              </ol>
            )}
          </div>
        ))}
        <div className="text-xs text-slate-500">
          Raw edge counts — upstream: {lineage.upstreamRaw}, downstream: {lineage.downstreamRaw}
        </div>
      </div>
    </div>
  );
}
