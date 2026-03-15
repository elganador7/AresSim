import { useSimStore } from "../../store/simStore";

export default function ViewSwitcher() {
  const activeView = useSimStore((s) => s.activeView);
  const setActiveView = useSimStore((s) => s.setActiveView);

  return (
    <div className="view-switcher">
      <button
        className={`view-btn view-btn-debug ${activeView === "debug" ? "view-btn-debug-active" : ""}`}
        onClick={() => setActiveView("debug")}
        title="Show all units (game master view)"
      >
        DEBUG
      </button>
      <button
        className={`view-btn view-btn-blue ${activeView === "blue" ? "view-btn-blue-active" : ""}`}
        onClick={() => setActiveView("blue")}
        title="Blue force view — your units only"
      >
        BLUE
      </button>
      <button
        className={`view-btn view-btn-red ${activeView === "red" ? "view-btn-red-active" : ""}`}
        onClick={() => setActiveView("red")}
        title="Red force view — your units only"
      >
        RED
      </button>
    </div>
  );
}
