package plugin_test

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

type fakePlugin struct {
	name string
}

func (f fakePlugin) Name() string              { return f.name }
func (f fakePlugin) CanParse(file string) bool { return true }
func (f fakePlugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	return nil, nil
}

func TestRegistry_RegisterAndLookup(t *testing.T) {
	plugin.Reset()
	defer plugin.Reset()

	plugin.Register(fakePlugin{name: "alpha"})
	plugin.Register(fakePlugin{name: "beta"})

	if names := plugin.Names(); len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("expected Names()=[alpha,beta] in registration order, got %v", names)
	}

	p, ok := plugin.Get("alpha")
	if !ok || p.Name() != "alpha" {
		t.Errorf("expected Get(alpha) to find fakePlugin, got %+v (ok=%v)", p, ok)
	}

	if _, ok := plugin.Get("missing"); ok {
		t.Error("expected Get(missing) to report not-found")
	}

	all := plugin.All()
	if len(all) != 2 {
		t.Errorf("expected All() to return 2 plugins, got %d", len(all))
	}
}

func TestRegistry_DuplicateNamePanics(t *testing.T) {
	plugin.Reset()
	defer plugin.Reset()

	plugin.Register(fakePlugin{name: "dup"})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected Register with a duplicate name to panic")
		}
	}()
	plugin.Register(fakePlugin{name: "dup"})
}
