// Package cache implements Repo Mapper's stateless-but-cached persistence:
// plain JSON files under .cache/ recording file hashes and previously
// parsed entities, so unchanged files are never reparsed (PRD section 16).
// Deliberately not a database — consistent with the "Stateless" design
// principle.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/anushibinj/repo-mapper/internal/model"
)

const (
	hashesFile   = "hashes.json"
	entitiesFile = "entities.json"
)

// Cache holds the on-disk cache state for one repository.
type Cache struct {
	dir string

	// Hashes maps file path -> content hash as of the last successful parse.
	Hashes map[string]string `json:"hashes"`
	// Entities maps file path -> entities produced by plugins for that file.
	Entities map[string][]model.Entity `json:"entities"`
}

// Dir returns the cache directory conventionally used under a repo root.
func Dir(repoRoot string) string {
	return filepath.Join(repoRoot, ".cache")
}

// Load reads the cache from dir, returning an empty-but-usable Cache if no
// cache exists yet.
func Load(dir string) (*Cache, error) {
	c := &Cache{
		dir:      dir,
		Hashes:   map[string]string{},
		Entities: map[string][]model.Entity{},
	}

	if data, err := os.ReadFile(filepath.Join(dir, hashesFile)); err == nil {
		_ = json.Unmarshal(data, &c.Hashes)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if data, err := os.ReadFile(filepath.Join(dir, entitiesFile)); err == nil {
		_ = json.Unmarshal(data, &c.Entities)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return c, nil
}

// Unchanged reports whether path's hash matches what is stored in the cache.
func (c *Cache) Unchanged(path, hash string) bool {
	if hash == "" {
		return false
	}
	stored, ok := c.Hashes[path]
	return ok && stored == hash
}

// Put records the hash and entities produced for a file.
func (c *Cache) Put(path, hash string, entities []model.Entity) {
	c.Hashes[path] = hash
	c.Entities[path] = entities
}

// Get returns previously cached entities for path, if any.
func (c *Cache) Get(path string) ([]model.Entity, bool) {
	e, ok := c.Entities[path]
	return e, ok
}

// Forget removes cache entries for a file that no longer exists.
func (c *Cache) Forget(path string) {
	delete(c.Hashes, path)
	delete(c.Entities, path)
}

// Save writes the cache to disk as JSON.
func (c *Cache) Save() error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(c.dir, hashesFile), c.Hashes); err != nil {
		return err
	}
	return writeJSON(filepath.Join(c.dir, entitiesFile), c.Entities)
}

// Clean removes the entire cache directory.
func Clean(dir string) error {
	return os.RemoveAll(dir)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
