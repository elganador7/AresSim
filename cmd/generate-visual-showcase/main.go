package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"

	"github.com/aressim/internal/scenario"
)

func main() {
	count := flag.Int("count", 256, "number of destroyers")
	columns := flag.Int("columns", 16, "number of columns in the grid")
	spacingKm := flag.Float64("spacing-km", 6, "spacing between ships in kilometers")
	outDir := flag.String("out-dir", "tmp/visual-showcase", "output directory")
	flag.Parse()

	scen := scenario.DestroyerVisualScaleScenario(*count, *columns, *spacingKm)
	raw, err := proto.Marshal(scen)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(err)
	}
	binPath := filepath.Join(*outDir, "destroyer-wall.binpb")
	b64Path := filepath.Join(*outDir, "destroyer-wall.b64")
	metaPath := filepath.Join(*outDir, "README.txt")
	if err := os.WriteFile(binPath, raw, 0o644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(b64Path, []byte(base64.StdEncoding.EncodeToString(raw)), 0o644); err != nil {
		panic(err)
	}
	readme := fmt.Sprintf(
		"Scenario: %s\nUnits: %d\nColumns: %d\nSpacingKm: %.2f\n\nFiles:\n- %s\n- %s\n",
		scen.GetName(), len(scen.GetUnits()), *columns, *spacingKm, binPath, b64Path,
	)
	if err := os.WriteFile(metaPath, []byte(readme), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s with %d destroyers\n", *outDir, len(scen.GetUnits()))
}

