package main

import "github.com/mattermost/mattermost-plugin-legal-hold/pluginmage"

func init() {
	// Register the processor binary
	pluginmage.RegisterBinary(pluginmage.BinaryBuildConfig{
		Name:             "processor",
		OutputPath:       "./bin",
		BinaryNameFormat: "processor-{{.Manifest.Version}}-{{.GOOS}}-{{.GOARCH}}",
		WorkingDir:       "./processor",
		PackagePath:      ".",
	})
}
