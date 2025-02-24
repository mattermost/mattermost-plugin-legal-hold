//go:build mage
// +build mage

// This file is maintained by the plugin sdk tooling.
// Please do not make changes to this file.

package main

import (
	//mage:import
	"github.com/mattermost/mattermost-plugin-legal-hold/plugin_magefile"
)

// Aliases defines some targets migrated from the old Makefile to ease the transition to mage
// You can find the default aliases in plugin_magefile/init.go
var Aliases = map[string]interface{}{
	"server":   plugin_magefile.Build.Server,
	"binaries": plugin_magefile.Build.AdditionalBinaries,
	"webapp":   plugin_magefile.Build.Webapp,
	"dist":     plugin_magefile.Build.All,
	"bundle":   plugin_magefile.Build.Bundle,
	"deploy":   plugin_magefile.Deploy.Upload,
}
