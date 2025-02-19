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
		OutputPath:  "./bin",
		BinaryName:  "processor",
		WorkingDir:  "./processor",
		PackagePath: ".",
	})
}

// Aliases defines some targets migrated from the old Makefile to ease the transition to mage
var Aliases = map[string]interface{}{
	"server":   plugin_magefile.Build.Server,
	"binaries": plugin_magefile.Build.AdditionalBinaries,
	"webapp":   plugin_magefile.Build.Webapp,
	"dist":     plugin_magefile.Build.All,
	"bundle":   plugin_magefile.Build.Bundle,
	"deploy":   plugin_magefile.Deploy.Upload,
}
