type Severity = "LOW" | "MEDIUM" | "HIGH" | "UNKNOWN";

const styles: Record<Severity, string> = {
  LOW: "bg-emerald-500/15 text-emerald-200 ring-1 ring-emerald-400/30",
  MEDIUM: "bg-amber-500/15 text-amber-200 ring-1 ring-amber-400/30",
  HIGH: "bg-rose-500/15 text-rose-200 ring-1 ring-rose-400/30",
  UNKNOWN: "bg-slate-500/15 text-slate-200 ring-1 ring-slate-400/30",
};

export function SeverityBadge({ severity }: { severity: Severity }) {
  return (
    <span className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold ${styles[severity]}`}>
      Severity: {severity}
    </span>
  );
}
