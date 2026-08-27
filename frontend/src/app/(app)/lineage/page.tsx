"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import dynamic from "next/dynamic";

const LineageGraphVis = dynamic(
  () => import("@/components/LineageGraphVis"),
  { ssr: false, loading: () => <p className="text-sm text-slate-500">Loading graph...</p> }
);

interface Node {
  id: string;
  type: string;
  name: string;
}

interface Edge {
  from: string;
  to: string;
}

export default function LineagePage() {
  const router = useRouter();
  const [graph, setGraph] = useState<{ nodes: Node[]; edges: Edge[] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .lineage()
      .then((data) => {
        setGraph(data);
        setLoading(false);
      })
      .catch((e) => {
        setError(e instanceof Error ? e.message : String(e));
        setLoading(false);
      });
  }, []);

  const handleNodeClick = (node: Node) => {
    const id = node.id;
    if (node.type === "data_source") {
      alert(`Data Source: ${node.name}\nID: ${id}`);
    } else if (node.type === "saved_query") {
      const queryId = id.replace("q:", "");
      router.push(`/queries/${queryId}`);
    } else if (node.type === "dashboard") {
      const dashId = id.replace("d:", "");
      router.push(`/dashboards/${dashId}`);
    } else {
      alert(`Node: ${node.name} (${node.type})`);
    }
  };

  if (loading) return <p className="text-sm text-slate-500">Loading lineage...</p>;
  if (error) return <p className="text-sm text-down">Error: {error}</p>;
  if (!graph || graph.nodes.length === 0) return <p className="text-sm text-slate-500">No lineage data found.</p>;

  return (
    <div className="mx-auto max-w-6xl">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-2xl font-extrabold tracking-tight">Data Lineage</h1>
        <div className="flex items-center gap-4">
          <span className="text-sm text-slate-400">
            {graph.nodes.length} nodes · {graph.edges.length} edges
          </span>
          <span className="text-xs text-slate-500">Click a node to navigate to the corresponding object.</span>
        </div>
      </div>

      <div className="glass rounded-2xl p-4">
        <LineageGraphVis data={graph} onNodeClick={handleNodeClick} />
        <div className="mt-4 flex flex-wrap gap-4 text-xs">
          <span className="flex items-center gap-1"><span className="w-3 h-3 rounded-full bg-[#4f7cff]"></span> Data Source</span>
          <span className="flex items-center gap-1"><span className="w-3 h-3 rounded-full bg-[#9b5de5]"></span> Saved Query</span>
          <span className="flex items-center gap-1"><span className="w-3 h-3 rounded-full bg-[#22d3c5]"></span> Dashboard</span>
          <span className="flex items-center gap-1"><span className="w-3 h-3 rounded-full bg-[#f5b942]"></span> Widget</span>
        </div>
      </div>
    </div>
  );
}
