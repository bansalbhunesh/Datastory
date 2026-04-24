import { useCallback, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

type Props = {
  title: string;
  subtitle?: string;
  markdown: string;
  source?: "claude" | "deterministic";
  warnings?: string[];
  filename?: string;
};

function translateWarning(w: string): string {
  if (w.includes("CLAUDE_API_KEY") || w.includes("AI enhancement not configured")) {
    return "AI enhancement not configured — showing verified rule-based report.";
  }
  if (w.includes("LLM rewrite failed") || w.includes("AI enhancement unavailable")) {
    return "AI enhancement unavailable — showing verified rule-based report.";
  }
  if (w.includes("lineage unavailable")) {
    return "Lineage data unavailable — severity may be underestimated.";
  }
  if (w.includes("lineage data could not be parsed") || w.includes("lineage parse failed")) {
    return "Lineage data could not be parsed — impact scope unknown.";
  }
  if (w.includes("data quality tests unavailable")) {
    return "Quality test data unavailable — root cause analysis is incomplete.";
  }
  return w;
}

export function ReportCard({ title, subtitle, markdown, source, warnings, filename }: Props) {
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState<string | null>(null);

  const onCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(markdown);
      setCopyError(null);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopyError("Clipboard permission denied. Use Download instead.");
      setCopied(false);
    }
  }, [markdown]);

  const onDownload = useCallback(() => {
    const blob = new Blob([markdown], { type: "text/markdown;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename ?? "datastory-incident-report.md";
    a.click();
    URL.revokeObjectURL(url);
  }, [markdown, filename]);

  return (
    <div className="rounded-2xl border border-slate-800 bg-gradient-to-b from-slate-900/70 to-slate-950/40 p-6 shadow-xl">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 flex-1">
          <div className="text-lg font-semibold text-white">{title}</div>
          {subtitle ? <div className="mt-1 truncate text-sm text-slate-400">{subtitle}</div> : null}
          {source ? (
            <div className="mt-2 flex flex-wrap gap-2">
              <span
                className={
                  source === "claude"
                    ? "rounded-full bg-indigo-500/15 px-2.5 py-0.5 text-xs font-semibold text-indigo-200 ring-1 ring-indigo-400/30"
                    : "rounded-full bg-slate-500/15 px-2.5 py-0.5 text-xs font-semibold text-slate-200 ring-1 ring-slate-400/25"
                }
                title={
                  source === "claude"
                    ? "Claude rewrote this report using only verified facts from OpenMetadata"
                    : "All facts sourced directly from OpenMetadata — no AI inference"
                }
              >
                {source === "claude" ? "✦ AI-Enhanced" : "◆ Rule-Based"}
              </span>
            </div>
          ) : null}
        </div>
        <div className="flex shrink-0 gap-2">
          <button
            type="button"
            onClick={onCopy}
            className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-100 hover:border-slate-500"
          >
            {copied ? "Copied" : "Copy"}
          </button>
          <button
            type="button"
            onClick={onDownload}
            className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-slate-100 hover:border-slate-500"
          >
            Download .md
          </button>
        </div>
      </div>

      {copyError ? <div className="mt-3 text-xs text-amber-300">{copyError}</div> : null}

      {warnings && warnings.length > 0 ? (
        <div className="mt-4 rounded-xl border border-amber-500/25 bg-amber-500/10 p-3 text-xs text-amber-100">
          <div className="font-semibold text-amber-200">Notice</div>
          <ul className="mt-2 list-disc space-y-1 pl-4">
            {warnings.map((w) => (
              <li key={w}>{translateWarning(w)}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <article className="prose prose-invert prose-sm mt-5 max-w-none rounded-xl border border-slate-800 bg-slate-950/40 p-5 prose-headings:scroll-mt-24 prose-a:text-indigo-300 prose-code:text-indigo-100">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{markdown}</ReactMarkdown>
      </article>
    </div>
  );
}
