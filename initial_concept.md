Here is the full content for your `README.md`. I’ve organized it to be "agent-ready," meaning if you feed this to an AI coding assistant, it will understand the architectural boundaries and the specific tech stack requirements immediately.

---

# Project Aegis: Operational & Strategic Wargame Engine

**Project Aegis** is a high-performance, local-first wargame engine designed for dual-use applications: complex strategic gaming and professional military operational planning. The engine prioritizes geospatial precision, deterministic simulation, and a "Common Operational Picture" (COP) interface.

## 🛠 Tech Stack

| Layer | Technology | Role |
| --- | --- | --- |
| **App Shell** | **Wails (Go)** | Native desktop performance & Go backend integration. |
| **Frontend** | **React & TypeScript** | Data-heavy UI components and dashboarding. |
| **Map Engine** | **CesiumJS** | 3D globe rendering with WGS84 ellipsoid precision. |
| **Database** | **SurrealDB (Embedded)** | Multi-model (Graph/Geo/Relational) local state. |
| **Data Protocol** | **Protobuf** | Binary serialization for cross-layer communication. |

---

## 📐 System Architecture

Aegis follows a **State-Sync** pattern. The Go backend serves as the "source of truth" (the Adjudicator), while the React frontend serves as the visualization layer.

1. **The Adjudicator (Go):** Manages the simulation loop, processes unit movement, and calculates combat outcomes using deterministic logic.
2. **The Persistence Layer (SurrealDB):** Stores unit positions as geospatial points and logistics/command structures as graph nodes.
3. **The Bridge (Protobuf/WebSockets):** Streams binary state updates to the frontend at high frequency, ensuring 60 FPS unit movement on the globe.

---

## 🛰 Protobuf Specification (`schema.proto`)

This is the single source of truth for the data model. All modifications to units or combat rules must start here.

```proto
syntax = "proto3";
package engine;

message Unit {
  string id = 1;
  string name = 2;
  string side = 3;             // Blue, Red, Neutral
  UnitType type = 4;
  Position position = 5;
  Capabilities capabilities = 6;
  Status status = 7;
  repeated Order orders = 8;   // Queue of operational tasks
}

enum UnitType {
  UNIT_TYPE_UNKNOWN = 0;
  UNIT_TYPE_INFANTRY = 1;
  UNIT_TYPE_ARMOR = 2;
  UNIT_TYPE_AIRCRAFT = 3;
  UNIT_TYPE_MARITIME = 4;
}

message Position {
  double lat = 1;              // WGS84 Decimal Degrees
  double lon = 2;
  double alt = 3;              // Meters above ellipsoid
  double heading = 4;          // True North orientation
}

message Capabilities {
  float max_speed = 1;         // Meters per second
  float kinetic_range = 2;     // Combat engagement radius (meters)
  float sensor_range = 3;      // Fog of War/Detection radius (meters)
}

message Status {
  float health = 1;            // 0.0 - 1.0
  float fuel_level = 2;
  bool is_active = 3;
}

```

---

## 🚀 MVP Roadmap

### **Phase 1: The Operational Core (Current)**

* [ ] Initialize Wails + React-TS skeleton.
* [ ] Embed SurrealDB in Go backend with geospatial indexing.
* [ ] Implement CesiumJS 3D globe in the main window.
* [ ] Protobuf-based "Spawn Unit" workflow (Go -> DB -> React/Cesium).

### **Phase 2: Logistics & Graph Logic**

* [ ] **Supply Lines:** Graph-based connectivity nodes where units require proximity to a "Logistics Link" to maintain combat effectiveness.
* [ ] **Dynamic Range Rings:** Real-time visual feedback for weapon and sensor envelopes on the globe.

### **Phase 3: Strategic Scaling**

* [ ] **Macro Simulation:** Manufacturing capacity, commodities infrastructure, and long-term attrition models.
* [ ] **Multiplayer:** Host-Client connectivity using WebSockets for collaborative planning.

---

## 🏗 Setup & Development

### Prerequisites

* **Go** (1.21+)
* **Node.js** (v18+)
* **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
* **Protobuf Compiler**: `protoc`

### Development Commands

```bash
# Install dependencies
cd frontend && npm install

# Run in development mode (hot-reloading)
wails dev

# Build production binary
wails build

```

---

**Would you like me to generate the actual `schema.proto` file and the Go initialization code for SurrealDB to get the project running?**