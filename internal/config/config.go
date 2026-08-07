// Package config loads repo-mapper.yaml and applies sane defaults so the
// tool works with zero configuration.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure, mirroring PRD section 19.
type Config struct {
	Scan         ScanConfig         `yaml:"scan"`
	Output       OutputConfig       `yaml:"output"`
	LLM          LLMConfig          `yaml:"llm"`
	Plugins      map[string]bool    `yaml:"plugins"`
	Autohandlers AutohandlerConfig  `yaml:"autohandlers"`
}

// AutohandlerConfig controls the optional post-generation hooks that update
// adjacent repository files (e.g. .github/copilot-instructions.md).
type AutohandlerConfig struct {
	// CopilotInstructions, when true (the default), writes or updates
	// .github/copilot-instructions.md after every scan/update so that
	// GitHub Copilot reads the repo-mapper entrypoint for context.
	CopilotInstructions bool `yaml:"copilot_instructions"`
}

// ScanConfig controls file discovery.
type ScanConfig struct {
	Exclude    []string `yaml:"exclude"`
	IgnoreFile string   `yaml:"ignoreFile"`
	Workers    int      `yaml:"workers"`
}

// OutputConfig controls where generated artifacts are written.
type OutputConfig struct {
	Directory string `yaml:"directory"`
}

// LLMConfig controls optional LLM-assisted summarisation.
type LLMConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
}

// DefaultFileName is the conventional config file name looked for in the
// repository root.
const DefaultFileName = "repo-mapper.yaml"

// Default returns a Config populated with sensible defaults, used both as a
// starting point before merging a config file and as the full config when
// no file is present.
func Default() *Config {
	return &Config{
		Scan: ScanConfig{
			Exclude: []string{
				"node_modules", "build", "dist", "target", ".git", ".repo-mapper", ".cache",
				"vendor", "bin", "obj", ".idea", ".vscode",
			},
			Workers: 0, // 0 => runtime.NumCPU() * 2, resolved by the scanner
		},
		Output: OutputConfig{
			Directory: ".repo-mapper",
		},
		LLM: LLMConfig{
			Enabled: false,
		},
		Plugins: map[string]bool{
			"java":   true,
			"spring": true,
			"react":  true,
			"vite":   true,
			"node":   true,
			"docker": true,
			"sql":    true,
		},
		Autohandlers: AutohandlerConfig{
			CopilotInstructions: true,
		},
	}
}

// Load reads repo-mapper.yaml from repoRoot if present, merging it over
// Default(). A missing file is not an error — Repo Mapper must work with
// zero configuration.
func Load(repoRoot string) (*Config, error) {
	cfg := Default()

	path := repoRoot + string(os.PathSeparator) + DefaultFileName
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// PluginEnabled reports whether the named plugin is enabled. Plugins not
// explicitly mentioned in config default to enabled.
func (c *Config) PluginEnabled(name string) bool {
	if c.Plugins == nil {
		return true
	}
	enabled, ok := c.Plugins[name]
	if !ok {
		return true
	}
	return enabled
}
