package model

import "testing"

func TestRepository_EqualIgnoringGit(t *testing.T) {
	base := &Repository{
		Name:      "demo",
		RootPath:  "/repo",
		Languages: []Language{{Name: "Java", FileCount: 3}},
		Modules:   []Module{{Name: "backend", Path: "backend", Kind: "backend"}},
		Git:       GitInfo{Branch: "master", CommitHash: "aaa111"},
	}

	t.Run("identical apart from commit hash is equal", func(t *testing.T) {
		other := &Repository{
			Name:      base.Name,
			RootPath:  base.RootPath,
			Languages: []Language{{Name: "Java", FileCount: 3}},
			Modules:   []Module{{Name: "backend", Path: "backend", Kind: "backend"}},
			Git:       GitInfo{Branch: "feature/x", CommitHash: "bbb222"},
		}
		if !base.EqualIgnoringGit(other) {
			t.Fatalf("expected repositories differing only in Git metadata to be equal")
		}
	})

	t.Run("real content difference is not equal", func(t *testing.T) {
		other := &Repository{
			Name:      base.Name,
			RootPath:  base.RootPath,
			Languages: []Language{{Name: "Java", FileCount: 4}}, // file count changed
			Modules:   []Module{{Name: "backend", Path: "backend", Kind: "backend"}},
			Git:       base.Git,
		}
		if base.EqualIgnoringGit(other) {
			t.Fatalf("expected repositories with differing content to not be equal")
		}
	})

	t.Run("nil handling", func(t *testing.T) {
		if base.EqualIgnoringGit(nil) {
			t.Fatalf("expected non-nil vs nil to be unequal")
		}
		var nilRepo *Repository
		if !nilRepo.EqualIgnoringGit(nil) {
			t.Fatalf("expected nil vs nil to be equal")
		}
	})
}
