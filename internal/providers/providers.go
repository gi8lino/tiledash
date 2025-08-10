package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/gi8lino/tiledash/internal/config"
)

// Runner is a compiled, ready-to-execute request bound to a provider.
type Runner interface {
	// Do performs the HTTP request (and pagination if enabled) and returns accumulator, pageCount, httpStatus.
	Do(ctx context.Context) (Accumulator, int, int, error)
}

// Registry maps provider names to HTTPProvider instances.
type Registry map[string]*HTTPProvider

// BuildRegistry constructs a Registry from config providers.
func BuildRegistry(cfg map[string]config.Provider) (Registry, error) {
	out := Registry{}
	for name, pc := range cfg {
		key := strings.ToLower(strings.TrimSpace(name))
		p, err := NewHTTPProvider(key, pc)
		if err != nil {
			return nil, err
		}
		out[key] = p
	}
	return out, nil
}

// BuildRunners compiles runners via Registry.Compile for each tile.
func BuildRunners(reg Registry, tiles []config.Tile) ([]Runner, error) {
	out := make([]Runner, len(tiles))
	for i := range tiles {
		r, err := reg.compile(tiles[i].Request)
		if err != nil {
			return nil, fmt.Errorf("tile %d (%s): %w", i, tiles[i].Title, err)
		}
		out[i] = r
	}
	return out, nil
}

// compile builds a Runner for a tile request using the registry.
func (r Registry) compile(req config.Request) (Runner, error) {
	name := strings.ToLower(strings.TrimSpace(req.Provider))
	p, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", req.Provider)
	}
	return p.NewRunner(req), nil
}
