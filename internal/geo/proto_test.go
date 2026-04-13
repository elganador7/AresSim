package geo

import (
	"testing"

	enginev1 "github.com/aressim/internal/gen/engine/v1"
)

func TestProtoPositionPopulatesH3(t *testing.T) {
	position := ProtoPosition(25.2854, 51.5310, 1500, 90, 250)
	if position.H3Cell == "" || position.H3ParentCell == "" {
		t.Fatalf("expected H3 fields to be populated: %+v", position)
	}
}

func TestPopulateProtoWaypointBackfillsH3(t *testing.T) {
	waypoint := &enginev1.Waypoint{Lat: 26.566, Lon: 56.25, AltMsl: 0}
	PopulateProtoWaypoint(waypoint)
	if waypoint.H3Cell == "" || waypoint.H3ParentCell == "" {
		t.Fatalf("expected H3 fields to be backfilled: %+v", waypoint)
	}
}

func TestPopulateProtoBatchUpdateBackfillsDeltaLocationFields(t *testing.T) {
	update := &enginev1.BatchUnitUpdate{
		Deltas: []*enginev1.UnitDelta{{
			UnitId: "u1",
			Position: &enginev1.Position{
				Lat: 25.2854, Lon: 51.5310, AltMsl: 1000,
			},
			MoveOrder: &enginev1.MoveOrder{
				Waypoints: []*enginev1.Waypoint{{Lat: 26.566, Lon: 56.25, AltMsl: 0}},
			},
		}},
	}
	PopulateProtoBatchUpdate(update)
	if update.Deltas[0].Position.GetH3Cell() == "" || update.Deltas[0].Position.GetH3ParentCell() == "" {
		t.Fatalf("expected delta position h3 fields to be populated")
	}
	if update.Deltas[0].MoveOrder.GetWaypoints()[0].GetH3Cell() == "" || update.Deltas[0].MoveOrder.GetWaypoints()[0].GetH3ParentCell() == "" {
		t.Fatalf("expected delta waypoint h3 fields to be populated")
	}
}
