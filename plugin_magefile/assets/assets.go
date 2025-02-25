package assets

import "embed"

//go:embed *.yml **/*/*.yml
var Assets embed.FS
