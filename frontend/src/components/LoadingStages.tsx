export type Stage = "idle" | "resolving" | "lineage" | "quality" | "writing" | "done";

const STAGES: { key: Stage; label: string }[] = [
  { key: "resolving", label: "Resolving table" },
  { key: "lineage", label: "Fetching lineage graph" },
  { key: "quality", label: "Analyzing quality tests" },
  { key: "writing", label: "Generating incident report" },
];

const ORDER = ["resolving", "lineage", "quality", "writing"];

export function LoadingStages({ stage }: { stage: Stage }) {
  if (stage === "idle" || stage === "done") return null;

  const currentIdx = ORDER.indexOf(stage);

  return (
    <div
      className="rounded-xl border border-slate-800 bg-slate-900/30 px-5 py-4"
      aria-live="polite"
      aria-label="Report generation progress"
    >
      <div className="space-y-2.5">
        {STAGES.map((s, idx) => {
          const isActive = s.key === stage;
          const isDone = idx < currentIdx;
          const isPending = idx > currentIdx;

          return (
            <div
              key={s.key}
              className={`flex items-center gap-3 text-sm transition-all duration-300 ${
                isActive
                  ? "text-indigo-200"
                  : isDone
                  ? "text-emerald-400"
                  : "text-slate-600"
              }`}
            >
              <div
                className={`h-2 w-2 flex-shrink-0 rounded-full transition-all duration-300 ${
                  isActive
                    ? "animate-pulse bg-indigo-400"
                    : isDone
                    ? "bg-emerald-400"
                    : "bg-slate-700"
                }`}
              />
              <span>{s.label}</span>
              {isActive && (
                <span className="animate-pulse text-indigo-400">...</span>
              )}
              {isDone && (
                <span className="text-xs text-emerald-400">✓</span>
              )}
              {isPending && null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
