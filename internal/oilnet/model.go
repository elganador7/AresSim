package oilnet

// Commodity is the material flowing through the oil network.
type Commodity string

const (
	CommodityCrude    Commodity = "crude"
	CommodityNGL      Commodity = "ngl"
	CommodityGasoline Commodity = "gasoline"
	CommodityDiesel   Commodity = "diesel"
	CommodityJet      Commodity = "jet_kerosene"
	CommodityFuelOil  Commodity = "fuel_oil"
	CommodityLPG      Commodity = "lpg"
	CommodityNaphtha  Commodity = "naphtha"
	CommodityRefinedProducts Commodity = "refined_products"
)

// NodeKind identifies the infrastructure role of a node.
type NodeKind string

const (
	NodeProject        NodeKind = "project"
	NodeExtractionSite NodeKind = "extraction_site"
	NodeGatheringHub   NodeKind = "gathering_hub"
	NodePipelineTerminal NodeKind = "pipeline_terminal"
	NodeExportTerminal NodeKind = "export_terminal"
	NodeImportTerminal NodeKind = "import_terminal"
	NodeRefinery       NodeKind = "refinery"
	NodeStorageHub     NodeKind = "storage_hub"
	NodeDemandCenter   NodeKind = "demand_center"
	NodeChokepoint     NodeKind = "chokepoint"
)

// EdgeKind identifies how oil moves between two nodes.
type EdgeKind string

const (
	EdgePipeline    EdgeKind = "pipeline"
	EdgeShipping    EdgeKind = "shipping_lane"
	EdgeInternalBus EdgeKind = "internal_transfer"
)

// OperationalState tracks infrastructure availability.
type OperationalState string

const (
	StateOperational OperationalState = "operational"
	StateDegraded    OperationalState = "degraded"
	StateOffline     OperationalState = "offline"
)

// SourceRef records provenance and confidence for an asset or flow datum.
type SourceRef struct {
	Name         string  `json:"name"`
	Organization string  `json:"organization"`
	URL          string  `json:"url"`
	LastUpdated  string  `json:"lastUpdated,omitempty"`
	Confidence   float64 `json:"confidence"`
	Notes        string  `json:"notes,omitempty"`
}

// ProductOutput describes refinery daily output for a derivative stream.
type ProductOutput struct {
	Commodity Commodity `json:"commodity"`
	BPD       float64   `json:"bpd"`
}

// CommodityQuantity represents an inventory or demand quantity for one commodity.
type CommodityQuantity struct {
	Commodity Commodity `json:"commodity"`
	BPD       float64   `json:"bpd,omitempty"`
	Barrels   float64   `json:"barrels,omitempty"`
}

// RoutePoint is a lon/lat vertex for transport infrastructure geometry.
type RoutePoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Node is a typed infrastructure node in the oil network graph.
type Node struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Kind               NodeKind            `json:"kind"`
	CountryCode        string              `json:"countryCode"`
	Operator           string              `json:"operator,omitempty"`
	Lat                float64             `json:"lat"`
	Lon                float64             `json:"lon"`
	State              OperationalState    `json:"state"`
	PrimaryCommodity   Commodity           `json:"primaryCommodity,omitempty"`
	ParentProjectID    string              `json:"parentProjectId,omitempty"`
	ChildFieldIDs      []string            `json:"childFieldIds,omitempty"`
	CapacityBPD        float64             `json:"capacityBpd,omitempty"`
	CurrentFlowBPD     float64             `json:"currentFlowBpd,omitempty"`
	SpareCapacityBPD   float64             `json:"spareCapacityBpd,omitempty"`
	ProductionBPD      float64             `json:"productionBpd,omitempty"`
	ReserveBbl         float64             `json:"reserveBbl,omitempty"`
	CrudeIntakeBPD     float64             `json:"crudeIntakeBpd,omitempty"`
	ProductOutputs     []ProductOutput     `json:"productOutputs,omitempty"`
	OutlineRings       [][]RoutePoint      `json:"outlineRings,omitempty"`
	DemandProfile      []CommodityQuantity `json:"demandProfile,omitempty"`
	Inventory          []CommodityQuantity `json:"inventory,omitempty"`
	DailyDrawLimitBPD  float64             `json:"dailyDrawLimitBpd,omitempty"`
	StorageCapacityBbl float64             `json:"storageCapacityBbl,omitempty"`
	Tags               []string            `json:"tags,omitempty"`
	Sources            []SourceRef         `json:"sources,omitempty"`
}

// Edge is a directional transport connection between two nodes.
type Edge struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Kind              EdgeKind         `json:"kind"`
	FromNodeID        string           `json:"fromNodeId"`
	ToNodeID          string           `json:"toNodeId"`
	Commodity         Commodity        `json:"commodity"`
	Commodities       []Commodity      `json:"commodities,omitempty"`
	CommodityLabel    string           `json:"commodityLabel,omitempty"`
	State             OperationalState `json:"state"`
	StatusDetail      string           `json:"statusDetail,omitempty"`
	CapacityBPD       float64          `json:"capacityBpd"`
	CurrentFlowBPD    float64          `json:"currentFlowBpd"`
	TransitDays       float64          `json:"transitDays,omitempty"`
	LengthKM          float64          `json:"lengthKm,omitempty"`
	Reversible        bool             `json:"reversible,omitempty"`
	CrossesChokepoint string           `json:"crossesChokepoint,omitempty"`
	CrossesChokepoints []string        `json:"crossesChokepoints,omitempty"`
	Route             []RoutePoint     `json:"route,omitempty"`
	Routes            [][]RoutePoint   `json:"routes,omitempty"`
	Sources           []SourceRef      `json:"sources,omitempty"`
}

// Graph is the full oil network graph presented to the UI.
type Graph struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Version     string      `json:"version"`
	View        string      `json:"view"`
	Nodes       []Node      `json:"nodes"`
	Edges       []Edge      `json:"edges"`
	Sources     []SourceRef `json:"sources,omitempty"`
}
