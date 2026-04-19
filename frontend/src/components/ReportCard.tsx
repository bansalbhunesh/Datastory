type Props = {
  title: string;
  subtitle?: string;
  markdown: string;
};

export function ReportCard({ title, subtitle, markdown }: Props) {
  return (
    <div className="rounded-2xl border border-slate-800 bg-gradient-to-b from-slate-900/70 to-slate-950/40 p-6 shadow-xl">
      <div className="flex flex-col gap-1">
        <div className="text-lg font-semibold text-white">{title}</div>
        {subtitle ? <div className="text-sm text-slate-400">{subtitle}</div> : null}
      </div>

      <div className="mt-5 whitespace-pre-wrap rounded-xl border border-slate-800 bg-slate-950/40 p-4 text-sm leading-6 text-slate-200">
        {markdown}
      </div>
    </div>
  );
}
