package plugin_magefile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Server mg.Namespace

// Build builds the server if it exists
func (Server) Build() error {
	if !info.Manifest.HasServer() {
		return nil
	}

	// Clean dist directory before creating it
	if err := sh.Rm(filepath.Join("server", "dist")); err != nil {
		return fmt.Errorf("failed to clean server/dist directory: %w", err)
	}

	// Create dist directory if it doesn't exist
	if err := os.MkdirAll(filepath.Join("server", "dist"), 0755); err != nil {
		return fmt.Errorf("failed to create server/dist directory: %w", err)
	}

	if info.EnableDeveloperMode {
		logger.Info("Building only for current platform due to MM_SERVICESETTINGS_ENABLEDEVELOPER",
			"namespace", "server",
			"target", "build",
			"GOOS", runtime.GOOS,
			"GOARCH", runtime.GOARCH)
		return buildServer(runtime.GOOS, runtime.GOARCH)
	}

	// Build for all supported platforms
	platforms := []struct{ GOOS, GOARCH string }{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
	}

	for _, p := range platforms {
		if err := buildServer(p.GOOS, p.GOARCH); err != nil {
			return err
		}
	}

	return nil
}

func buildServer(goos, goarch string) error {
	logger.Info("Building server",
		"namespace", "server",
		"target", "build",
		"GOOS", goos,
		"GOARCH", goarch)

	// Prepare build args
	buildArgs := []string{
		"build",
		"-trimpath",
	}

	// Add build flags if set
	if info.GoBuildFlags != "" {
		buildArgs = append(buildArgs, info.GoBuildFlags)
	}

	// Add gcflags if set
	if info.GoBuildGcflags != "" {
		buildArgs = append(buildArgs, "-gcflags", info.GoBuildGcflags)
	}

	// Add output and package
	buildArgs = append(buildArgs,
		"-o", filepath.Join("server", "dist", "plugin-"+goos+"-"+goarch),
	)

	cmd := NewCmd("server", "build", map[string]string{
		"GOOS":        goos,
		"GOARCH":      goarch,
		"CGO_ENABLED": "0",
	})
	cmd.WorkingDir("server")
	if err := cmd.Run("go", buildArgs...); err != nil {
		return fmt.Errorf("failed to build server for %s/%s: %w", goos, goarch, err)
	}

	return nil
}
