package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScan_DiscoversAndHashesFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "Main.java"), "package main;\nclass Main {}\n")
	writeFile(t, filepath.Join(root, "node_modules", "pkg", "index.js"), "module.exports = {};\n")
	writeFile(t, filepath.Join(root, "README.md"), "# hi\n")

	// node_modules is not ignored by the scanner itself (only .gitignore
	// entries and .git are, unconditionally) — callers (the CLI) are
	// expected to pass config.Default().Exclude. Exercise that same
	// contract here rather than relying on an implicit default.
	files, err := Scan(Options{RepoRoot: root, Exclude: []string{"node_modules"}})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}

	assertContains(t, paths, "src/Main.java")
	assertContains(t, paths, "README.md")
	assertNotContains(t, paths, "node_modules/pkg/index.js")

	for _, f := range files {
		if f.Path == "src/Main.java" {
			if f.Hash == "" {
				t.Error("expected non-empty hash for Main.java")
			}
			if f.Language != "Java" {
				t.Errorf("expected Language=Java, got %q", f.Language)
			}
		}
	}
}

func TestScan_HonoursGitignore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "build/\n*.log\n")
	writeFile(t, filepath.Join(root, "build", "out.txt"), "x")
	writeFile(t, filepath.Join(root, "debug.log"), "x")
	writeFile(t, filepath.Join(root, "keep.txt"), "x")

	files, err := Scan(Options{RepoRoot: root})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	assertContains(t, paths, "keep.txt")
	assertNotContains(t, paths, "build/out.txt")
	assertNotContains(t, paths, "debug.log")
}

func TestScan_DeterministicOrder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "b.txt"), "b")
	writeFile(t, filepath.Join(root, "a.txt"), "a")
	writeFile(t, filepath.Join(root, "c.txt"), "c")

	files, err := Scan(Options{RepoRoot: root})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if files[0].Path != "a.txt" || files[1].Path != "b.txt" || files[2].Path != "c.txt" {
		t.Errorf("expected sorted order a,b,c; got %s,%s,%s", files[0].Path, files[1].Path, files[2].Path)
	}
}

func TestClassifyLanguage(t *testing.T) {
	cases := map[string]string{
		"Foo.java":       "Java",
		"index.tsx":      "TypeScript",
		"App.jsx":        "JavaScript",
		"schema.sql":     "SQL",
		"Dockerfile":     "Docker",
		"package.json":   "Node",
		"unknown.xyz123": "",
	}
	for path, want := range cases {
		if got := ClassifyLanguage(path); got != want {
			t.Errorf("ClassifyLanguage(%q) = %q, want %q", path, got, want)
		}
	}
}

func assertContains(t *testing.T, list []string, want string) {
	t.Helper()
	for _, v := range list {
		if v == want {
			return
		}
	}
	t.Errorf("expected list to contain %q, got %v", want, list)
}

func assertNotContains(t *testing.T, list []string, unwanted string) {
	t.Helper()
	for _, v := range list {
		if v == unwanted {
			t.Errorf("expected list NOT to contain %q, got %v", unwanted, list)
		}
	}
}
