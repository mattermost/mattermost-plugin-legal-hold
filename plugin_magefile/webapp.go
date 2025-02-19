package plugin_magefile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

type Webapp mg.Namespace

// InstallDeps installs webapp dependencies using npm
func (Webapp) InstallDeps() error {
	if !info.Manifest.HasWebapp() {
		return nil
	}

	nodeModulesPath := filepath.Join("webapp", "node_modules")
	packageJSONPath := filepath.Join("webapp", "package.json")

	// Check if node_modules is newer than package.json
	newer, err := target.Path(nodeModulesPath, packageJSONPath)
	if err != nil {
		return fmt.Errorf("failed to check dependencies: %w", err)
	}
	if !newer {
		return nil // node_modules is up to date
	}

	logger.Info("Installing webapp dependencies",
		"namespace", "webapp",
		"target", "installdeps")

	cmd := NewCmd("webapp", "installdeps", nil)
	if err := cmd.WorkingDir("webapp").Run("npm", "install"); err != nil {
		return fmt.Errorf("failed to install webapp dependencies: %w", err)
	}

	return nil
}

// Build builds the webapp if it exists
func (Build) Webapp() error {
	mg.Deps(Webapp.InstallDeps)

	if !info.Manifest.HasWebapp() {
		return nil
	}

	cmd := NewCmd("webapp", "build", nil)

	// Clean dist directory before creating it
	if err := sh.Rm(filepath.Join("webapp", "dist")); err != nil {
		return fmt.Errorf("failed to clean webapp/dist directory: %w", err)
	}

	// Create dist directory if it doesn't exist
	if err := os.MkdirAll(filepath.Join("webapp", "dist"), 0755); err != nil {
		return fmt.Errorf("failed to create webapp/dist directory: %w", err)
	}

	logger.Info("Building webapp",
		"namespace", "webapp",
		"target", "build")

	if err := cmd.WorkingDir("webapp").Run("npm", "run", "build"); err != nil {
		return fmt.Errorf("failed to build webapp: %w", err)
	}

	return nil
}
