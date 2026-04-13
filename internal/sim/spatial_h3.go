package sim

import (
	"github.com/aressim/internal/geo"
	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

type unitH3Index map[string][]*enginev1.Unit

const coarseRangeH3Resolution = geo.AggregateH3Resolution

func maxDetectorSensorRange(def DefStats) float64 {
	best := def.DetectionRangeM
	for _, sensor := range def.SensorSuite {
		if sensor.MaxRangeM > best {
			best = sensor.MaxRangeM
		}
	}
	return best
}

func buildUnitH3Index(units []*enginev1.Unit) unitH3Index {
	index := make(unitH3Index)
	for _, unit := range units {
		if unit == nil || unit.GetPosition() == nil {
			continue
		}
		geo.PopulateProtoPosition(unit.GetPosition())
		cell := unit.GetPosition().GetH3ParentCell()
		if cell == "" {
			continue
		}
		index[cell] = append(index[cell], unit)
	}
	return index
}

func candidateUnitsInRange(detector *enginev1.Unit, maxRangeM float64, index unitH3Index, allUnits []*enginev1.Unit) []*enginev1.Unit {
	if detector == nil || detector.GetPosition() == nil || maxRangeM <= 0 {
		return allUnits
	}
	geo.PopulateProtoPosition(detector.GetPosition())
	cell := detector.GetPosition().GetH3ParentCell()
	if cell == "" {
		return allUnits
	}
	cells, err := geo.GridDiskForRange(geo.H3Cell(cell), maxRangeM, coarseRangeH3Resolution)
	if err != nil {
		return allUnits
	}
	seen := make(map[string]bool)
	candidates := make([]*enginev1.Unit, 0)
	for _, neighbor := range cells {
		for _, unit := range index[string(neighbor)] {
			if unit == nil || seen[unit.GetId()] {
				continue
			}
			seen[unit.GetId()] = true
			candidates = append(candidates, unit)
		}
	}
	if len(candidates) == 0 {
		return allUnits
	}
	return candidates
}
