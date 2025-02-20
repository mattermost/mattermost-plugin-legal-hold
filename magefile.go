//go:build mage

package main

import (
	//mage:import
	"github.com/mattermost/mattermost-plugin-legal-hold/plugin_magefile"
)

// RegisterAdditionalBinaries allows plugins to register additional binaries to be built
func init() {
	// Register the processor binary
	plugin_magefile.RegisterBinary(plugin_magefile.BinaryBuildConfig{
		Name:             "processor",
		OutputPath:       "./bin",
		BinaryNameFormat: "processor-{{.Manifest.Version}}-{{.GOOS}}-{{.GOARCH}}",
		WorkingDir:       "./processor",
		PackagePath:      ".",
	})
}

// Aliases defines some targets migrated from the old Makefile to ease the transition to mage
// You can find the default aliases in plugin_magefile/init.go
var Aliases = map[string]interface{}{
	"server":   plugin_magefile.Build.Server,
	"binaries": plugin_magefile.Build.AdditionalBinaries,
	"webapp":   plugin_magefile.Webapp.Watch,
	"dist":     plugin_magefile.Build.All,
	"bundle":   plugin_magefile.Build.Bundle,
	"deploy":   plugin_magefile.Deploy.Upload,
}
