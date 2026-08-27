"use client";

import { useEffect, useRef } from "react";
import { Network } from "vis-network/standalone/esm/vis-network";

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

interface LineageGraphVisProps {
  data: GraphData;
  onNodeClick?: (node: Node) => void;
}

const typeColors: Record<string, string> = {
  data_source: "#4f7cff",
  saved_query: "#9b5de5",
  dashboard: "#22d3c5",
  widget: "#f5b942",
};

export default function LineageGraphVis({ data, onNodeClick }: LineageGraphVisProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const networkRef = useRef<any>(null);

  useEffect(() => {
    if (!containerRef.current || !data || data.nodes.length === 0) return;

    const nodes = data.nodes.map((n) => ({
      id: n.id,
      label: n.name,
      color: typeColors[n.type] || "#8b93a7",
      title: `${n.type}: ${n.name}`,
    }));

    const edges = data.edges.map((e) => ({
      id: `${e.from}->${e.to}`,
      from: e.from,
      to: e.to,
      arrows: "to",
      color: { color: "rgba(255,255,255,0.3)" },
    }));

    const options = {
      layout: {
        hierarchical: false,
      },
      physics: {
        enabled: true,
        stabilization: { iterations: 100 },
      },
      interaction: {
        hover: true,
        tooltipDelay: 200,
      },
      nodes: {
        shape: "dot",
        size: 20,
        font: {
          color: "#e6e9f0",
          size: 14,
          face: "sans-serif",
        },
        borderWidth: 0,
      },
      edges: {
        width: 2,
        smooth: {
          enabled: true,
          type: "continuous",
          roundness: 0,
        },
      },
    };

    const network = new Network(containerRef.current, { nodes, edges }, options as any);
    networkRef.current = network;

    network.on("click", (params) => {
      if (params.nodes.length > 0) {
        const nodeId = params.nodes[0];
        const original = data.nodes.find((n) => n.id === nodeId);
        if (original && onNodeClick) {
          onNodeClick(original);
        }
      }
    });

    setTimeout(() => network.fit(), 100);

    return () => {
      if (networkRef.current) {
        networkRef.current.destroy();
        networkRef.current = null;
      }
    };
  }, [data, onNodeClick]);

  return <div ref={containerRef} className="w-full h-[600px] rounded-xl overflow-hidden border border-white/10 bg-base-900/50" />;
}
