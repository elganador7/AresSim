package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aressim/internal/library"
	"gopkg.in/yaml.v3"
)

type outputFile struct {
	Library     library.Meta         `yaml:"library"`
	Definitions []library.Definition `yaml:"definitions"`
}

func main() {
	root := filepath.Join("internal", "library", "data", "default")

	files, err := collectYAML(root)
	if err != nil {
		fail(err)
	}

	grouped := make(map[string][]library.Definition)
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}

		var file library.File
		if err := yaml.Unmarshal(raw, &file); err != nil {
			fail(fmt.Errorf("parse %s: %w", path, err))
		}
		for _, def := range file.Definitions {
			key := filepath.Join(domainDir(def.Domain), outputFileName(def))
			grouped[key] = append(grouped[key], def)
		}
	}

	keys := sortedKeys(grouped)
	for _, relPath := range keys {
		defs := grouped[relPath]
		sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })

		out := outputFile{
			Library: library.Meta{
				ID:          strings.TrimSuffix(filepath.Base(relPath), ".yaml") + "-v1",
				Name:        humanizeFileName(filepath.Base(relPath)),
				Description: fmt.Sprintf("%s units originating from %s.", titleWords(strings.ReplaceAll(domainDir(defs[0].Domain), "_", " ")), countryLabel(defs[0].NationOfOrigin)),
				Version:     "1.0.0",
				Author:      "AresSim Default Library",
			},
			Definitions: defs,
		}

		raw, err := yaml.Marshal(&out)
		if err != nil {
			fail(fmt.Errorf("marshal %s: %w", relPath, err))
		}

		outPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			fail(err)
		}
		if err := os.WriteFile(outPath, raw, 0o644); err != nil {
			fail(err)
		}
	}

	for _, path := range files {
		if err := os.Remove(path); err != nil {
			fail(err)
		}
	}

	if err := removeEmptyDirs(root); err != nil {
		fail(err)
	}
}

func collectYAML(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files, err
}

func domainDir(domain int) string {
	switch domain {
	case 1:
		return "land"
	case 2:
		return "air"
	case 3:
		return "sea"
	case 4:
		return "subsurface"
	default:
		return "other"
	}
}

func outputFileName(def library.Definition) string {
	return fmt.Sprintf("%s_%s_units.yaml", strings.ToLower(def.NationOfOrigin), domainDir(def.Domain))
}

func humanizeFileName(name string) string {
	base := strings.TrimSuffix(name, ".yaml")
	base = strings.ReplaceAll(base, "_", " ")
	parts := strings.Fields(base)
	for i, part := range parts {
		if strings.EqualFold(part, "usa") || strings.EqualFold(part, "uk") || len(part) == 3 {
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

func countryLabel(code string) string {
	if strings.TrimSpace(code) == "" {
		return "unknown origin"
	}
	return strings.ToUpper(code)
}

func titleWords(s string) string {
	parts := strings.Fields(s)
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

func sortedKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])
	})
	return keys
}

func removeEmptyDirs(root string) error {
	var dirs []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return err
	}

	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, dir := range dirs {
		if dir == root {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			if err := os.Remove(dir); err != nil {
				return err
			}
		}
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
