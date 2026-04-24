import { useMemo } from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  Cell,
  ResponsiveContainer,
} from "recharts";
import type { IncidentEntry } from "../api/incidents";

const SEV_NUM: Record<string, number> = { LOW: 1, MEDIUM: 2, HIGH: 3 };
const SEV_COLOR: Record<string, string> = {
  LOW: "#34d399",
  MEDIUM: "#fbbf24",
  HIGH: "#f87171",
};

type ChartPoint = {
  time: string;
  severity: number;
  severityLabel: string;
  color: string;
};

type TooltipProps = {
  active?: boolean;
  payload?: Array<{ payload: ChartPoint }>;
};

function CustomTooltip({ active, payload }: TooltipProps) {
  if (!active || !payload || payload.length === 0) return null;
  const d = payload[0].payload;
  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-xs shadow-xl">
      <div
        className="font-bold"
        style={{ color: d.color }}
      >
        {d.severityLabel}
      </div>
      <div className="mt-0.5 text-slate-400">{d.time}</div>
    </div>
  );
}

type Props = { incidents: IncidentEntry[] };

export function IncidentChart({ incidents }: Props) {
  const data = useMemo<ChartPoint[]>(
    () =>
      [...incidents]
        .reverse()
        .map((inc) => ({
          time: new Date(inc.createdAt * 1000).toLocaleString(undefined, {
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          }),
          severity: SEV_NUM[inc.severity] ?? 1,
          severityLabel: inc.severity,
          color: SEV_COLOR[inc.severity] ?? SEV_COLOR.LOW,
        })),
    [incidents]
  );

  if (data.length < 2) return null;

  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-4">
      <div className="text-sm font-semibold text-slate-200">Severity trend</div>
      <div className="mt-0.5 text-xs text-slate-500">
        {data.length} incident{data.length !== 1 ? "s" : ""} recorded
      </div>
      <div className="mt-3" style={{ height: 80 }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 2, right: 2, bottom: 0, left: -28 }}>
            <XAxis
              dataKey="time"
              tick={false}
              axisLine={false}
              tickLine={false}
            />
            <YAxis
              domain={[0, 3]}
              ticks={[1, 2, 3]}
              tickFormatter={(v) => ["", "L", "M", "H"][v as number] ?? ""}
              tick={{ fontSize: 9, fill: "#64748b" }}
              axisLine={false}
              tickLine={false}
              width={20}
            />
            <Tooltip content={<CustomTooltip />} cursor={{ fill: "#1e293b" }} />
            <Bar dataKey="severity" radius={[3, 3, 0, 0]} maxBarSize={24}>
              {data.map((d, i) => (
                <Cell key={i} fill={d.color} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
      <div className="mt-2 flex gap-3 text-[10px] text-slate-500">
        <span>
          <span className="text-emerald-400">■</span> LOW
        </span>
        <span>
          <span className="text-amber-400">■</span> MEDIUM
        </span>
        <span>
          <span className="text-rose-400">■</span> HIGH
        </span>
      </div>
    </div>
  );
}
