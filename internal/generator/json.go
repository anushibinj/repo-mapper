package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/anushibinj/repo-mapper/internal/model"
)

// WriteJSON renders repo-map.json, feature-map.json, api-map.json,
// ownership-map.json, and routes.json (PRD section 13, "JSON").
func WriteJSON(repo *model.Repository, outputDir string) error {
	if err := writeJSONFile(filepath.Join(outputDir, "repo-map.json"), repo); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outputDir, "feature-map.json"), repo.Features); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outputDir, "api-map.json"), repo.APIs); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outputDir, "ownership-map.json"), buildOwnershipMap(repo)); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outputDir, "routes.json"), repo.Routes); err != nil {
		return err
	}
	return nil
}

// ownershipEntry maps one component to the feature that claims it, if any.
type ownershipEntry struct {
	Component string `json:"component"`
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Feature   string `json:"feature,omitempty"`
}

func buildOwnershipMap(repo *model.Repository) []ownershipEntry {
	owner := map[string]string{}
	for _, f := range repo.Features {
		for _, name := range f.Frontend {
			owner[name] = f.Name
		}
		for _, name := range f.Backend {
			owner[name] = f.Name
		}
	}

	entries := make([]ownershipEntry, 0, len(repo.Components))
	for _, c := range repo.Components {
		entries = append(entries, ownershipEntry{
			Component: c.Name,
			Kind:      c.Kind,
			File:      c.File,
			Feature:   owner[c.Name],
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Component < entries[j].Component })
	return entries
}

func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
