// Package docspack is a small example format pack consumed by the Fuego docs
// site itself, so the documentation dogfoods the pack system it describes.
//
// It contributes the site's tags taxonomy and tutorials collection as config
// defaults (deep-merged under config.yaml — run `fuego config` to see the
// provenance) and ships a "page-meta" partial that renders a page's tags.
package docspack

import (
	_ "embed"

	"github.com/FabioSol/fuego/core"
)

//go:embed config-defaults.yaml
var configDefaults []byte

// Pack returns the docs example pack.
func Pack() core.Pack {
	return core.Pack{
		Name:           "docs",
		ConfigDefaults: configDefaults,
	}
}
