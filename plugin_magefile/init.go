package plugin_magefile

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

var (
	info   *pluginInfo
	Logger *slog.Logger

	DefaultAliases = map[string]interface{}{
		"server":   Build.Server,
		"binaries": Build.AdditionalBinaries,
		"webapp":   Webapp.Watch,
		"dist":     Build.All,
		"bundle":   Build.Bundle,
		"deploy":   Deploy.Upload,
	}
)

// initializeEnvironment performs all the setup checks previously done in setup.mk
func initializeEnvironment() error {
	// Check if go is installed
	if !CheckCommand("go") {
		return fmt.Errorf("go is not available: see https://golang.org/doc/install")
	}

	// Check if tar is GNU tar
	cmd := exec.Command("tar", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("tar is not available")
	}
	if !strings.Contains(string(output), "GNU tar") {
		return fmt.Errorf("GNU tar is required but system tar is not GNU compatible. Please install GNU tar on your system")
	}

	// Initialize plugin info
	info = &pluginInfo{}
	info.Init()
	info.Defaults()

	// Parse plugin.json
	manifestBytes, err := os.ReadFile("plugin.json")
	if err != nil {
		return fmt.Errorf("failed to read plugin.json: %w", err)
	}

	info.Manifest = &model.Manifest{}
	if err := json.Unmarshal(manifestBytes, info.Manifest); err != nil {
		return fmt.Errorf("failed to parse plugin.json: %w", err)
	}

	// Check if webapp is defined
	if info.Manifest.HasWebapp() {
		// If webapp exists, verify npm is installed
		if _, err := exec.LookPath("npm"); err != nil {
			return fmt.Errorf("npm is not available: see https://www.npmjs.com/get-npm")
		}
	}

	// Setup the plugin binary configuration in the build pipeline
	setupPluginBinary()

	return nil
}

// init runs the initialization when the package is imported
func init() {
	// Initialize logger with custom handler
	Logger = slog.New(NewCustomHandler(os.Stdout))

	if err := initializeEnvironment(); err != nil {
		Logger.Error("Error initializing environment", "error", err)
		os.Exit(1)
	}
}
