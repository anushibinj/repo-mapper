// Package scanner discovers files in a repository, honours .gitignore and a
// custom ignore file, classifies files by language, and computes content
// hashes — all in parallel (PRD section 9).
package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// File describes one discovered, non-ignored file.
type File struct {
	// Path is relative to the repository root, using forward slashes.
	Path     string
	AbsPath  string
	Size     int64
	Hash     string
	Language string
	Binary   bool
}

// Options configures a scan.
type Options struct {
	// RepoRoot is the absolute path to the repository root.
	RepoRoot string
	// Exclude is a list of directory/file name globs to always skip,
	// evaluated in addition to .gitignore.
	Exclude []string
	// IgnoreFile is an additional gitignore-syntax file to honour, relative
	// to RepoRoot (e.g. ".repomapperignore"). Optional.
	IgnoreFile string
	// Workers is the number of parallel hashing workers. 0 => NumCPU * 2.
	Workers int
}

// Scan walks RepoRoot and returns all non-ignored, non-binary-by-default
// files with their language classification and content hash. Results are
// returned sorted by Path for determinism.
func Scan(opts Options) ([]File, error) {
	root := opts.RepoRoot
	matcher := newIgnoreMatcher(root, opts.Exclude, opts.IgnoreFile)

	var candidates []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // fault tolerant: skip unreadable entries
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		if matcher.matches(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}
		candidates = append(candidates, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan string, len(candidates))
	results := make(chan File, len(candidates))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range jobs {
				results <- processFile(root, rel)
			}
		}()
	}
	for _, c := range candidates {
		jobs <- c
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	files := make([]File, 0, len(candidates))
	for f := range results {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func processFile(root, rel string) File {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	f := File{Path: rel, AbsPath: abs, Language: ClassifyLanguage(rel)}

	info, err := os.Stat(abs)
	if err == nil {
		f.Size = info.Size()
	}

	binary, hash, err := hashFile(abs)
	if err != nil {
		return f
	}
	f.Binary = binary
	f.Hash = hash
	return f
}

// HashFile computes the content hash of a single file, honouring the same
// binary-sniffing rule as Scan. Exposed for incremental update workflows
// that need to hash individual changed files without a full tree walk.
func HashFile(absPath string) (string, error) {
	_, hash, err := hashFile(absPath)
	return hash, err
}

func hashFile(abs string) (binary bool, hash string, err error) {
	fh, err := os.Open(abs)
	if err != nil {
		return false, "", err
	}
	defer fh.Close()

	if isBinary(fh) {
		return true, "", nil
	}

	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return false, "", err
	}
	return false, hex.EncodeToString(h.Sum(nil)), nil
}

// isBinary does a cheap sniff of the first 8KB for NUL bytes, resetting the
// file position afterwards.
func isBinary(f *os.File) bool {
	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	_, _ = f.Seek(0, io.SeekStart)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

var languageByExt = map[string]string{
	".java":       "Java",
	".kt":         "Kotlin",
	".go":         "Go",
	".py":         "Python",
	".rb":         "Ruby",
	".cs":         "C#",
	".php":        "PHP",
	".rs":         "Rust",
	".js":         "JavaScript",
	".jsx":        "JavaScript",
	".mjs":        "JavaScript",
	".cjs":        "JavaScript",
	".ts":         "TypeScript",
	".tsx":        "TypeScript",
	".sql":        "SQL",
	".yml":        "YAML",
	".yaml":       "YAML",
	".json":       "JSON",
	".xml":        "XML",
	".html":       "HTML",
	".css":        "CSS",
	".scss":       "SCSS",
	".md":         "Markdown",
	".dockerfile": "Docker",
}

// ClassifyLanguage returns a human-readable language name for a file path
// based on its extension/name, or "" if unknown.
func ClassifyLanguage(relPath string) string {
	base := filepath.Base(relPath)
	if strings.EqualFold(base, "dockerfile") || strings.HasPrefix(strings.ToLower(base), "dockerfile.") {
		return "Docker"
	}
	if strings.EqualFold(base, "package.json") {
		return "Node"
	}
	ext := strings.ToLower(filepath.Ext(relPath))
	if lang, ok := languageByExt[ext]; ok {
		return lang
	}
	return ""
}
