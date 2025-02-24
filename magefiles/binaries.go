package main

import "github.com/mattermost/mattermost-plugin-legal-hold/plugin_magefile"

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
