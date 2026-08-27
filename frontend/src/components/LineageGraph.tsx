"use client";

import { useEffect, useRef } from "react";
import { ForceGraph2D } from "react-force-graph";

interface Node {
  id: string;
  name: string;
  type: string;
}

interface Edge {
  from: string;
  to: string;
}

interface GraphData {
  nodes: Node[];
  edges: Edge[];
}

interface LineageGraphProps {
  data: GraphData;
  onNodeClick?: (node: Node) => void;
}

const typeColors: Record<string, string> = {
  data_source: "#4f7cff",
  saved_query: "#9b5de5",
  dashboard: "#22d3c5",
  widget: "#f5b942",
};

export default function LineageGraph({ data, onNodeClick }: LineageGraphProps) {
  const graphRef = useRef<any>();

  useEffect(() => {
    if (graphRef.current) {
      graphRef.current.d3Force("charge")?.strength(-300);
      graphRef.current.d3Force("link")?.distance(200);
      graphRef.current.zoomToFit(0);
    }
  }, [data]);

  if (!data || data.nodes.length === 0) {
    return <p className="text-sm text-slate-500">No lineage data to display.</p>;
  }

  const graphData = {
    nodes: data.nodes.map((n) => ({
      id: n.id,
      name: n.name,
      type: n.type,
      color: typeColors[n.type] || "#8b93a7",
    })),
    links: data.edges.map((e) => ({
      source: e.from,
      target: e.to,
    })),
  };

  return (
    <div className="w-full h-[600px] rounded-xl overflow-hidden border border-white/10 bg-base-900/50">
      <ForceGraph2D
        ref={graphRef}
        graphData={graphData}
        nodeLabel="name"
        nodeColor="color"
        nodeVal={12}
        linkWidth={2}
        linkColor={() => "rgba(255,255,255,0.3)"}
        onNodeClick={(node: any) => {
          if (onNodeClick) {
            const original = data.nodes.find((n) => n.id === node.id);
            if (original) onNodeClick(original);
          }
        }}
        nodeCanvasObject={(node: any, ctx: CanvasRenderingContext2D, globalScale: number) => {
          const label = node.name;
          const fontSize = 10 / globalScale;
          ctx.font = `${fontSize}px sans-serif`;
          ctx.fillStyle = "#e6e9f0";
          ctx.textAlign = "center";
          ctx.textBaseline = "bottom";
          ctx.fillText(label, node.x, node.y - 10);
        }}
      />
    </div>
  );
}
