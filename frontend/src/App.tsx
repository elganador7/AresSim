import { useEffect, useState } from "react";
import { initBridge } from "./bridge/bridge";
import CesiumGlobe from "./components/CesiumGlobe";
import ScenarioEditor from "./components/editor/ScenarioEditor";
import EventLog from "./components/hud/EventLog";
import TopBar from "./components/hud/TopBar";
import UnitPanel from "./components/hud/UnitPanel";
import ViewSwitcher from "./components/hud/ViewSwitcher";
import { RequestSync } from "../wailsjs/go/main/App";
import "./app.css";

export default function App() {
  const [appView, setAppView] = useState<"sim" | "editor">("sim");

  useEffect(() => {
    initBridge();
    RequestSync().catch((e) => console.warn("[App] RequestSync:", e));
    console.log("[App] AresSim frontend initialized");
  }, []);

  if (appView === "editor") {
    return (
      <ScenarioEditor
        onExit={() => setAppView("sim")}
        onPlay={() => setAppView("sim")}
      />
    );
  }

  return (
    <div className="app-shell">
      <CesiumGlobe />

      <div className="hud-overlay">
        <TopBar onOpenEditor={() => setAppView("editor")} />
        <ViewSwitcher />
        <EventLog />
        <UnitPanel />
      </div>
    </div>
  );
}
