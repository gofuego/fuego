package cli

import (
	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

// packLayers extracts config-default layers from registered packs, in
// registration order (lowest precedence first).
func packLayers(packs []core.Pack) ([]config.Layer, error) {
	var layers []config.Layer
	for _, p := range packs {
		if len(p.ConfigDefaults) == 0 {
			continue
		}
		layer, err := config.ParsePackLayer(p.Name, p.ConfigDefaults)
		if err != nil {
			return nil, err
		}
		layers = append(layers, layer)
	}
	return layers, nil
}

// loadConfig loads the user config merged with pack config defaults.
func loadConfig(configPath string, packs []core.Pack) (*config.Config, error) {
	layers, err := packLayers(packs)
	if err != nil {
		return nil, err
	}
	cfg, _, err := config.LoadLayered(configPath, layers)
	return cfg, err
}
