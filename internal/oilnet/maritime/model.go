package maritime

import "github.com/aressim/internal/oilnet"

type TerminalType string

const (
	TerminalTypeCrudeExport   TerminalType = "crude_export"
	TerminalTypeCrudeImport   TerminalType = "crude_import"
	TerminalTypeProductExport TerminalType = "product_export"
	TerminalTypeProductImport TerminalType = "product_import"
	TerminalTypeMixed         TerminalType = "mixed"
)

type TankerClass string

const (
	TankerClassVLCC    TankerClass = "vlcc"
	TankerClassSuezmax TankerClass = "suezmax"
	TankerClassAframax TankerClass = "aframax"
	TankerClassLR2     TankerClass = "lr2"
	TankerClassLR1     TankerClass = "lr1"
	TankerClassMR      TankerClass = "mr"
	TankerClassLPG     TankerClass = "lpg"
)

type CargoInference string

const (
	CargoInferenceCrude           CargoInference = "crude"
	CargoInferenceRefinedProducts CargoInference = "refined_products"
	CargoInferenceLPG             CargoInference = "lpg"
	CargoInferenceNGL             CargoInference = "ngl"
	CargoInferenceUnknown         CargoInference = "unknown"
)

type MarineTerminal struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	CountryCode        string                  `json:"countryCode"`
	Operator           string                  `json:"operator,omitempty"`
	Lat                float64                 `json:"lat"`
	Lon                float64                 `json:"lon"`
	State              oilnet.OperationalState `json:"state"`
	TerminalType       TerminalType            `json:"terminalType"`
	ProductsHandled    []oilnet.Commodity      `json:"productsHandled,omitempty"`
	StorageCapacityBbl float64                 `json:"storageCapacityBbl,omitempty"`
	CapacityBPD        float64                 `json:"capacityBpd,omitempty"`
	BerthCount         int                     `json:"berthCount,omitempty"`
	DraftClass         string                  `json:"draftClass,omitempty"`
	Tags               []string                `json:"tags,omitempty"`
	Sources            []oilnet.SourceRef      `json:"sources,omitempty"`
}

type CanalTransit struct {
	ID               string                  `json:"id"`
	Name             string                  `json:"name"`
	NodeID           string                  `json:"nodeId"`
	CountryCode      string                  `json:"countryCode"`
	Lat              float64                 `json:"lat"`
	Lon              float64                 `json:"lon"`
	State            oilnet.OperationalState `json:"state"`
	CapacityBPD      float64                 `json:"capacityBpd,omitempty"`
	TypicalDelayDays float64                 `json:"typicalDelayDays,omitempty"`
	Sources          []oilnet.SourceRef      `json:"sources,omitempty"`
}

type SeaborneCorridor struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	FromTerminalID     string                  `json:"fromTerminalId"`
	ToTerminalID       string                  `json:"toTerminalId"`
	Commodity          oilnet.Commodity        `json:"commodity"`
	ProductsHandled    []oilnet.Commodity      `json:"productsHandled,omitempty"`
	State              oilnet.OperationalState `json:"state"`
	VesselClass        TankerClass             `json:"vesselClass,omitempty"`
	TypicalCargoBbl    float64                 `json:"typicalCargoBbl,omitempty"`
	CapacityBPD        float64                 `json:"capacityBpd,omitempty"`
	CurrentFlowBPD     float64                 `json:"currentFlowBpd,omitempty"`
	TransitDays        float64                 `json:"transitDays,omitempty"`
	LengthKM           float64                 `json:"lengthKm,omitempty"`
	CrossesChokepoints []string                `json:"crossesChokepoints,omitempty"`
	Route              []oilnet.RoutePoint     `json:"route,omitempty"`
	Sources            []oilnet.SourceRef      `json:"sources,omitempty"`
}

type Voyage struct {
	ID             string         `json:"id"`
	IMO            string         `json:"imo,omitempty"`
	MMSI           string         `json:"mmsi,omitempty"`
	VesselName     string         `json:"vesselName,omitempty"`
	TankerClass    TankerClass    `json:"tankerClass,omitempty"`
	CargoInference CargoInference `json:"cargoInference,omitempty"`
	FromTerminalID string         `json:"fromTerminalId,omitempty"`
	ToTerminalID   string         `json:"toTerminalId,omitempty"`
}

type MaritimeFlowSnapshot struct {
	ID        string             `json:"id"`
	Date      string             `json:"date"`
	Corridors []SeaborneCorridor `json:"corridors"`
	Terminals []MarineTerminal   `json:"terminals"`
	Canals    []CanalTransit     `json:"canals,omitempty"`
	Sources   []oilnet.SourceRef `json:"sources,omitempty"`
}

type Topology struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Version     string             `json:"version"`
	Terminals   []MarineTerminal   `json:"terminals"`
	Canals      []CanalTransit     `json:"canals"`
	Corridors   []SeaborneCorridor `json:"corridors"`
	Sources     []oilnet.SourceRef `json:"sources,omitempty"`
}
