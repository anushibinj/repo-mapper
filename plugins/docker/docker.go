// Package docker parses Dockerfile and docker-compose files to surface
// infrastructure services, images, and exposed ports.
package docker

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
	"gopkg.in/yaml.v3"
)

func init() {
	plugin.Register(New())
}

// Plugin is the Docker plugin.
type Plugin struct{}

// New constructs a Docker Plugin.
func New() *Plugin { return &Plugin{} }

// Name implements plugin.Plugin.
func (p *Plugin) Name() string { return "docker" }

// CanParse implements plugin.Plugin.
func (p *Plugin) CanParse(file string) bool {
	base := strings.ToLower(filepath.Base(file))
	if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") {
		return true
	}
	return base == "docker-compose.yml" || base == "docker-compose.yaml" ||
		base == "compose.yml" || base == "compose.yaml"
}

var (
	fromRe   = regexp.MustCompile(`(?mi)^\s*FROM\s+([^\s]+)`)
	exposeRe = regexp.MustCompile(`(?mi)^\s*EXPOSE\s+([\d\s]+)`)
)

type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image     string   `yaml:"image"`
	Build     any      `yaml:"build"`
	Ports     []string `yaml:"ports"`
	DependsOn any      `yaml:"depends_on"`
}

// Parse implements plugin.Plugin.
func (p *Plugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	base := strings.ToLower(filepath.Base(ctx.RelPath))
	if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") {
		return p.parseDockerfile(ctx, content)
	}
	return p.parseCompose(ctx, content)
}

func (p *Plugin) parseDockerfile(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	src := string(content)
	attrs := map[string]string{}
	if m := fromRe.FindStringSubmatch(src); m != nil {
		attrs["baseImage"] = m[1]
	}
	if m := exposeRe.FindStringSubmatch(src); m != nil {
		attrs["exposedPorts"] = strings.Join(strings.Fields(m[1]), ",")
	}
	return []model.Entity{{
		Kind:       "docker-image",
		Name:       filepath.Dir(ctx.RelPath),
		File:       ctx.RelPath,
		Language:   "Docker",
		Attributes: attrs,
	}}, nil
}

func (p *Plugin) parseCompose(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	var cf composeFile
	if err := yaml.Unmarshal(content, &cf); err != nil {
		return nil, err
	}

	var entities []model.Entity
	for name, svc := range cf.Services {
		attrs := map[string]string{
			"image": svc.Image,
			"ports": strings.Join(svc.Ports, ","),
		}
		var refs []string
		switch d := svc.DependsOn.(type) {
		case []any:
			for _, v := range d {
				if s, ok := v.(string); ok {
					refs = append(refs, s)
				}
			}
		case map[string]any:
			for k := range d {
				refs = append(refs, k)
			}
		}
		entities = append(entities, model.Entity{
			Kind:       "docker-service",
			Name:       name,
			File:       ctx.RelPath,
			Language:   "Docker",
			Attributes: attrs,
			Refs:       refs,
		})
	}
	return entities, nil
}
