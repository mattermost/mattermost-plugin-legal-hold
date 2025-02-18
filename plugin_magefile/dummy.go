package plugin_magefile

import "time"

// Dummy prints a hello message and plugin info
func Dummy() error {
	// The plugin info is already initialized via init() when this runs
	logger.Debug("Debug")
	logger.Info("Info")
	logger.Warn("Warn")
	logger.Error("Error")
	time.Sleep(1 * time.Second)
	logger.Info("Plugin info",
		"id", info.Manifest.Id,
		"version", info.Manifest.Version,
		"name", info.Manifest.Name)
	return nil
}
