import { useEffect, useRef } from "react";
import { useSimStore } from "../../store/simStore";

const categoryColor: Record<string, string> = {
  combat: "#ef4444",
  logistics: "#f59e0b",
  c2: "#3b82f6",
  intelligence: "#a855f7",
  scenario: "#6b7280",
};

export default function EventLog() {
  const eventLog = useSimStore((s) => s.eventLog);
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [eventLog.length]);

  if (eventLog.length === 0) {
    return (
      <div className="event-log">
        <div className="event-log-header">EVENT LOG</div>
        <div className="event-log-empty">Awaiting simulation events…</div>
      </div>
    );
  }

  return (
    <div className="event-log">
      <div className="event-log-header">
        EVENT LOG <span className="event-count">({eventLog.length})</span>
      </div>
      <div className="event-log-entries">
        {eventLog.map((entry) => (
          <div key={entry.id} className="event-entry">
            <span
              className="entry-category"
              style={{ color: categoryColor[entry.category] ?? "#6b7280" }}
            >
              [{entry.category.toUpperCase()}]
            </span>{" "}
            <span className="entry-text">{entry.text}</span>
          </div>
        ))}
        <div ref={endRef} />
      </div>
    </div>
  );
}
