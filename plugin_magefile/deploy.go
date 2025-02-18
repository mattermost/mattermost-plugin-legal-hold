package plugin_magefile

import (
	"github.com/magefile/mage/mg"
)

type Deploy mg.Namespace

// Upload builds and installs the plugin to a server
func (Deploy) Upload() error {
	mg.SerialDeps(Dist.Build, Pluginctl.Deploy)

	return nil
}
