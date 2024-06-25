package config

import "github.com/mattermost/mattermost/server/public/model"

// Configuration captures the plugin's external Configuration as exposed in the Mattermost server
// Configuration, as well as values computed from the Configuration. Any public fields will be
// deserialized from the Mattermost server Configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// Configuration can change at any time, access to the Configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the Configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your Configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type Configuration struct {
	TimeOfDay              string
	AmazonS3BucketSettings AmazonS3BucketSettings
}

type AmazonS3BucketSettings struct {
	Enable   bool
	Settings model.FileSettings
}

// Clone shallow copies the Configuration. Your implementation may require a deep copy if
// your Configuration has reference types.
func (c *Configuration) Clone() *Configuration {
	var clone = *c
	return &clone
}
