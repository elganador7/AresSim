import { useMemo } from "react";
import { useShallow } from "zustand/react/shallow";
import { useSimStore } from "../../store/simStore";
import { selectOilPanelState } from "../../store/simSelectors";

function formatBpd(value?: number): string {
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value ?? 0);
}

function formatBarrels(value?: number): string {
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value ?? 0);
}

function formatCommodity(value?: string): string {
  return String(value ?? "")
    .replaceAll("_", " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
}

function formatSourceBlend(sources: Array<{ organization?: string; name?: string }>): string {
  const labels = Array.from(new Set(
    sources
      .map((source) => source.organization || source.name || "")
      .map((value) => value.trim())
      .filter(Boolean),
  ));
  return labels.length > 0 ? labels.join(", ") : "N/A";
}

export default function OilPanel() {
  const {
    oilGraph,
    selectedOilNodeId,
    selectedOilEdgeId,
    selectOilNode,
    selectOilEdge,
  } = useSimStore(useShallow(selectOilPanelState));

  const detail = useMemo(() => {
    if (!oilGraph) {
      return null;
    }
    if (selectedOilNodeId) {
      const node = oilGraph.nodes.find((entry) => entry.id === selectedOilNodeId);
      if (!node) {
        return null;
      }
      return {
        kind: "node" as const,
        title: node.name,
        subtitle: `${formatCommodity(node.kind)} · ${node.countryCode || "UNK"}`,
        state: node.state,
        rows: [
          ["Current Flow", `${formatBpd(node.currentFlowBpd)} bpd`],
          ["Production", `${formatBpd(node.productionBpd)} bpd`],
          ["Reserves", `${formatBarrels(node.reserveBbl)} bbl`],
          ["Capacity", `${formatBpd(node.capacityBpd)} bpd`],
          ["Spare Capacity", `${formatBpd(node.spareCapacityBpd)} bpd`],
          ["Primary Commodity", formatCommodity(node.primaryCommodity)],
          ["Operator", node.operator || "Unknown"],
          ["Source Blend", formatSourceBlend(node.sources ?? [])],
          ["Fields", node.childFieldIds?.length ? `${node.childFieldIds.length}` : "N/A"],
        ],
        outputs: node.productOutputs ?? [],
        sources: node.sources ?? [],
      };
    }
    if (selectedOilEdgeId) {
      const edge = oilGraph.edges.find((entry) => entry.id === selectedOilEdgeId);
      if (!edge) {
        return null;
      }
      return {
        kind: "edge" as const,
        title: edge.name,
        subtitle: `${formatCommodity(edge.kind)} · ${edge.commodityLabel || formatCommodity(edge.commodity)}`,
        state: edge.state,
        rows: [
          ["Current Flow", `${formatBpd(edge.currentFlowBpd)} bpd`],
          ["Capacity", `${formatBpd(edge.capacityBpd)} bpd`],
          ["Products", edge.commodityLabel || (edge.commodities?.length ? edge.commodities.map(formatCommodity).join(", ") : formatCommodity(edge.commodity))],
          ["Transit", edge.transitDays ? `${edge.transitDays.toFixed(1)} days` : "N/A"],
          ["Length", edge.lengthKm ? `${Math.round(edge.lengthKm).toLocaleString()} km` : "N/A"],
          ["Chokepoint", edge.crossesChokepoint || "None"],
          ["Source Blend", formatSourceBlend(edge.sources ?? [])],
        ],
        outputs: [],
        sources: edge.sources ?? [],
      };
    }
    return null;
  }, [oilGraph, selectedOilEdgeId, selectedOilNodeId]);

  if (!detail) {
    return null;
  }

  return (
    <aside className="oil-panel">
      <div className="oil-panel-header">
        <div>
          <div className="oil-panel-title">Oil Network</div>
          <div className="oil-panel-name">{detail.title}</div>
          <div className="oil-panel-subtitle">{detail.subtitle}</div>
        </div>
        <button
          className="unit-panel-close"
          onClick={() => {
            selectOilNode(null);
            selectOilEdge(null);
          }}
          title="Close"
        >
          ×
        </button>
      </div>
      <div className="oil-panel-body">
        <div className={`oil-panel-state ${detail.state}`}>
          {detail.state.toUpperCase()}
        </div>
        {detail.rows.map(([label, value]) => (
          <div className="oil-panel-row" key={label}>
            <span className="oil-panel-label">{label}</span>
            <span className="oil-panel-value">{value || "N/A"}</span>
          </div>
        ))}
        {detail.outputs.length > 0 && (
          <div className="oil-panel-section">
            <div className="oil-panel-section-title">Refined Output</div>
            {detail.outputs.map((output) => (
              <div className="oil-panel-row" key={output.commodity}>
                <span className="oil-panel-label">{formatCommodity(output.commodity)}</span>
                <span className="oil-panel-value">{formatBpd(output.bpd)} bpd</span>
              </div>
            ))}
          </div>
        )}
        {detail.sources.length > 0 && (
          <div className="oil-panel-section">
            <div className="oil-panel-section-title">Sources</div>
            {detail.sources.slice(0, 4).map((source) => (
              <div className="oil-panel-source" key={`${source.organization}:${source.name}`}>
                <span>{source.organization || source.name}</span>
                <span>{Math.round((source.confidence ?? 0) * 100)}%</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}
