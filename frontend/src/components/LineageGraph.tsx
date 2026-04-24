import { useMemo, useCallback } from "react";
import ReactFlow, {
  type Node,
  type Edge,
  type NodeTypes,
  Handle,
  Position,
  MarkerType,
  Background,
  BackgroundVariant,
} from "reactflow";
import "reactflow/dist/style.css";
import type { LineageSummary } from "../types/report";

function shortLabel(fqn: string, maxParts = 3): string {
  const parts = fqn.split(".");
  return parts.slice(-Math.min(maxParts, parts.length)).join(".");
}

type TableNodeData = {
  fqn: string;
  label: string;
  kind: "upstream" | "focal" | "downstream";
};

function TableNode({ data }: { data: TableNodeData }) {
  const cls =
    data.kind === "focal"
      ? "bg-indigo-500/20 border-indigo-400/70 text-indigo-100 shadow-lg shadow-indigo-500/10"
      : "bg-slate-800/80 border-slate-600/60 text-slate-200";

  return (
    <div
      className={`rounded-lg border px-3 py-2 text-xs ${cls}`}
      style={{ minWidth: 110, maxWidth: 185 }}
      title={data.fqn}
    >
      <Handle
        type="target"
        position={Position.Left}
        style={{ opacity: 0, pointerEvents: "none" }}
      />
      <div className="truncate font-medium">{data.label}</div>
      {data.kind === "focal" && (
        <div className="mt-0.5 text-[10px] text-indigo-300/70">focal table</div>
      )}
      <Handle
        type="source"
        position={Position.Right}
        style={{ opacity: 0, pointerEvents: "none" }}
      />
    </div>
  );
}

const nodeTypes: NodeTypes = { table: TableNode };

const LEVEL_X = { upstream: 0, focal: 260, downstream: 520 };
const NODE_GAP = 58;

export function LineageGraph({ lineage }: { lineage: LineageSummary }) {
  const { nodes, edges } = useMemo(() => {
    const nodes: Node<TableNodeData>[] = [];
    const edges: Edge[] = [];

    // Center everything around y=0
    const upCount = lineage.upstream.length;
    const downCount = lineage.downstream.length;
    const focalY =
      Math.max(upCount, downCount) > 0
        ? ((Math.max(upCount, downCount) - 1) * NODE_GAP) / 2
        : 0;

    nodes.push({
      id: "focal",
      type: "table",
      position: { x: LEVEL_X.focal, y: focalY },
      data: { fqn: lineage.focal, label: shortLabel(lineage.focal), kind: "focal" },
    });

    lineage.upstream.forEach((fqn, i) => {
      const id = `up-${i}`;
      nodes.push({
        id,
        type: "table",
        position: { x: LEVEL_X.upstream, y: i * NODE_GAP },
        data: { fqn, label: shortLabel(fqn), kind: "upstream" },
      });
      edges.push({
        id: `e-${id}`,
        source: id,
        target: "focal",
        style: { stroke: "#64748b", strokeWidth: 1.5 },
        markerEnd: { type: MarkerType.ArrowClosed, color: "#64748b", width: 14, height: 14 },
      });
    });

    lineage.downstream.forEach((fqn, i) => {
      const id = `down-${i}`;
      nodes.push({
        id,
        type: "table",
        position: { x: LEVEL_X.downstream, y: i * NODE_GAP },
        data: { fqn, label: shortLabel(fqn), kind: "downstream" },
      });
      edges.push({
        id: `e-${id}`,
        source: "focal",
        target: id,
        animated: true,
        style: { stroke: "#6366f1", strokeWidth: 1.5 },
        markerEnd: { type: MarkerType.ArrowClosed, color: "#6366f1", width: 14, height: 14 },
      });
    });

    return { nodes, edges };
  }, [lineage]);

  // eslint-disable-next-line @typescript-eslint/no-empty-function
  const noop = useCallback(() => {}, []);

  const isEmpty =
    lineage.upstream.length === 0 &&
    lineage.downstream.length === 0 &&
    !lineage.focal;

  if (isEmpty) {
    return (
      <div className="flex h-32 items-center justify-center rounded-xl border border-slate-800 bg-slate-900/40 text-xs text-slate-500">
        No lineage data available
      </div>
    );
  }

  const height =
    Math.max(
      lineage.upstream.length,
      lineage.downstream.length,
      1
    ) *
      NODE_GAP +
    80;

  return (
    <div
      className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900/40"
      style={{ height: Math.min(Math.max(height, 140), 300) }}
    >
      <div className="px-3 pt-2.5 text-xs font-semibold text-slate-300">
        Lineage Graph
        <span className="ml-2 font-normal text-slate-500">
          {lineage.upstream.length} upstream · {lineage.downstream.length} downstream
        </span>
      </div>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={noop}
        onEdgesChange={noop}
        fitView
        fitViewOptions={{ padding: 0.25 }}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        panOnDrag={false}
        zoomOnScroll={false}
        preventScrolling={false}
        proOptions={{ hideAttribution: true }}
        style={{ background: "transparent" }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          color="#1e293b"
          gap={20}
          size={1}
        />
      </ReactFlow>
    </div>
  );
}
