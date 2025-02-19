package plugin_magefile

import (
	"github.com/magefile/mage/mg"
)

type Deploy mg.Namespace

// Upload builds and installs the plugin to a server
func (Deploy) Upload() error {
	mg.SerialDeps(Build.All, Pluginctl.Deploy)

	return nil
}
