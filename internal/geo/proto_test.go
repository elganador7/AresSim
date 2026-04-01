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
