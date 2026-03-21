import { useEffect, useState } from "react";
import { initBridge } from "./bridge/bridge";
import CesiumGlobe from "./components/CesiumGlobe";
import ScenarioEditor from "./components/editor/ScenarioEditor";
import EventLog from "./components/hud/EventLog";
import ScenarioLoadModal from "./components/hud/ScenarioLoadModal";
import TopBar from "./components/hud/TopBar";
import UnitPanel from "./components/hud/UnitPanel";
import { useSimStore } from "./store/simStore";
import { RequestSync } from "../wailsjs/go/main/App";
import "./app.css";

function MapModeBanner() {
  const mapCommandMode = useSimStore((s) => s.mapCommandMode);
  const units = useSimStore((s) => s.units);

  if (mapCommandMode.type === "target_pick" && mapCommandMode.unitId) {
    const shooter = units.get(mapCommandMode.unitId);
    return (
      <div className="map-mode-banner">
        Target Pick Mode
        <span className="map-mode-banner-detail">Click an enemy unit for {shooter?.displayName ?? "selected unit"}</span>
      </div>
    );
  }

  if (mapCommandMode.type === "route" && mapCommandMode.unitId) {
    const mover = units.get(mapCommandMode.unitId);
    return (
      <div className="map-mode-banner">
        Route Edit Mode
        <span className="map-mode-banner-detail">Click the map to append waypoints for {mover?.displayName ?? "selected unit"}</span>
      </div>
    );
  }

  return null;
}

export default function App() {
  const [appView, setAppView] = useState<"sim" | "editor">("sim");
  const [scenarioLoadOpen, setScenarioLoadOpen] = useState(false);
  const [debugViewMenuVisible, setDebugViewMenuVisible] = useState(false);

  useEffect(() => {
    initBridge();
    RequestSync().catch((e) => console.warn("[App] RequestSync:", e));
    console.log("[App] AresSim frontend initialized");
  }, []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.metaKey && event.key.toLowerCase() === "d") {
        event.preventDefault();
        setDebugViewMenuVisible((current) => !current);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
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
        <TopBar
          onOpenEditor={() => setAppView("editor")}
          onOpenScenario={() => setScenarioLoadOpen(true)}
          debugViewMenuVisible={debugViewMenuVisible}
        />
        <MapModeBanner />
        <EventLog />
        <UnitPanel />
        <ScenarioLoadModal open={scenarioLoadOpen} onClose={() => setScenarioLoadOpen(false)} />
      </div>
    </div>
  );
}
