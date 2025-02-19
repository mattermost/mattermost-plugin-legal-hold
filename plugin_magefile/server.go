package plugin_magefile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
)

// Build builds the server if it exists
func (Build) Server() error {
	if !info.Manifest.HasServer() {
		return nil
	}

	// Validate all binary configurations before starting the build
	for _, config := range AllBinaries {
		if err := config.IsValid(); err != nil {
			return fmt.Errorf("invalid build configuration for binary '%s': %w", config.BinaryName, err)
		}
	}

	// Clean dist directory before creating it
	if err := sh.Rm(filepath.Join("server", "dist")); err != nil {
		return fmt.Errorf("failed to clean server/dist directory: %w", err)
	}

	// Create dist directory if it doesn't exist
	if err := os.MkdirAll(filepath.Join("server", "dist"), 0755); err != nil {
		return fmt.Errorf("failed to create server/dist directory: %w", err)
	}

	pluginBinary := AllBinaries[0]

	for _, platform := range pluginBinary.Platforms {
		if err := buildBinary(pluginBinary, platform); err != nil {
			return fmt.Errorf("failed to build %s for %s/%s: %w",
				pluginBinary.BinaryName, platform.GOOS, platform.GOARCH, err)
		}
	}

	return nil
}

// AdditionalBinaries builds all additional binaries if set up by the plugin developers
func (Build) AdditionalBinaries() error {
	for _, config := range AllBinaries[1:] {
		for _, platform := range config.Platforms {
			if err := buildBinary(config, platform); err != nil {
				return fmt.Errorf("failed to build %s for %s/%s: %w",
					config.BinaryName, platform.GOOS, platform.GOARCH, err)
			}
		}
	}

	return nil
}

func buildBinary(config BinaryBuildConfig, platform BuildPlatform) error {
	Logger.Info("Building binary",
		"namespace", "server",
		"target", "build",
		"binary", config.BinaryName,
		"GOOS", platform.GOOS,
		"GOARCH", platform.GOARCH)

	// Prepare build args
	buildArgs := append([]string{"build"}, DefaultBuildFlags...)

	// Add build flags if set
	if config.GoBuildFlags != "" {
		buildArgs = append(buildArgs, config.GoBuildFlags)
	}

	// Add gcflags if set
	if config.GcFlags != "" {
		buildArgs = append(buildArgs, "-gcflags", config.GcFlags)
	}

	// Add output and package
	outputName := fmt.Sprintf("%s-%s-%s-%s", config.BinaryName, info.Manifest.Version, platform.GOOS, platform.GOARCH)
	if platform.GOOS == "windows" {
		outputName += ".exe"
	}

	buildArgs = append(buildArgs,
		"-o", filepath.Join(config.OutputPath, outputName),
		config.PackagePath,
	)

	// Set up environment
	env := map[string]string{
		"GOOS":   platform.GOOS,
		"GOARCH": platform.GOARCH,
	}
	// Merge with config environment
	for k, v := range config.Environment {
		env[k] = v
	}

	cmd := NewCmd("server", "build", env)
	if config.WorkingDir != "" {
		cmd.WorkingDir(config.WorkingDir)
	}

	if err := cmd.Run("go", buildArgs...); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}

	return nil
}
