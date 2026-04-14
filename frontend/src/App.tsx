import { Suspense, lazy, useEffect, useState } from "react";
import { initBridge } from "./bridge/bridge";
import CesiumGlobe from "./components/CesiumGlobe";
import EventLog from "./components/hud/EventLog";
import OilPanel from "./components/hud/OilPanel";
import TargetPanel from "./components/hud/TargetPanel";
import TopBar from "./components/hud/TopBar";
import UnitPanel from "./components/hud/UnitPanel";
import { useSimStore } from "./store/simStore";
import { reportError } from "./utils/errors";
import { RequestSync } from "../wailsjs/go/main/App";
import "./app.css";

const ScenarioEditor = lazy(() => import("./components/editor/ScenarioEditor"));
const ScenarioLoadModal = lazy(() => import("./components/hud/ScenarioLoadModal"));

function MapModeBanner() {
  const mapCommandMode = useSimStore((s) => s.mapCommandMode);
  const units = useSimStore((s) => s.units);

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
    RequestSync().catch((error) => reportError("App:RequestSync", error));
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
      <Suspense fallback={<div className="app-shell" />}>
        <ScenarioEditor
          onExit={() => setAppView("sim")}
          onPlay={() => setAppView("sim")}
        />
      </Suspense>
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
        <OilPanel />
        <UnitPanel />
        <TargetPanel />
        <Suspense fallback={null}>
          <ScenarioLoadModal open={scenarioLoadOpen} onClose={() => setScenarioLoadOpen(false)} />
        </Suspense>
      </div>
    </div>
  );
}
