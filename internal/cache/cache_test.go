package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anushibinj/repo-mapper/internal/model"
)

func TestLoad_EmptyDirReturnsUsableCache(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".cache")
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if c.Unchanged("foo.java", "abc") {
		t.Error("expected Unchanged=false for a never-seen file")
	}
	if _, ok := c.Get("foo.java"); ok {
		t.Error("expected Get to report not-found for a never-seen file")
	}
}

func TestCache_PutSaveLoadRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".cache")
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	entities := []model.Entity{{Kind: "java-type", Name: "Foo", File: "Foo.java", Language: "Java"}}
	c.Put("Foo.java", "hash123", entities)

	if err := c.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "hashes.json")); err != nil {
		t.Errorf("expected hashes.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "entities.json")); err != nil {
		t.Errorf("expected entities.json to exist: %v", err)
	}

	c2, err := Load(dir)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if !c2.Unchanged("Foo.java", "hash123") {
		t.Error("expected Foo.java to be reported unchanged after reload")
	}
	got, ok := c2.Get("Foo.java")
	if !ok || len(got) != 1 || got[0].Name != "Foo" {
		t.Errorf("expected cached entity for Foo.java to round-trip, got %+v (ok=%v)", got, ok)
	}
}

func TestCache_ForgetRemovesEntry(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".cache")
	c, _ := Load(dir)
	c.Put("Foo.java", "hash123", []model.Entity{{Name: "Foo"}})
	c.Forget("Foo.java")

	if c.Unchanged("Foo.java", "hash123") {
		t.Error("expected Unchanged=false after Forget")
	}
	if _, ok := c.Get("Foo.java"); ok {
		t.Error("expected Get to report not-found after Forget")
	}
}

func TestClean_RemovesCacheDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".cache")
	c, _ := Load(dir)
	c.Put("Foo.java", "hash123", nil)
	if err := c.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := Clean(dir); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected cache dir to be removed, stat err=%v", err)
	}
}
