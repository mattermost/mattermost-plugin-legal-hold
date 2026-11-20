package config

import "github.com/mattermost/mattermost-server/v6/model"

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
const (
	// FilestoreDriverServer uses the Mattermost server file store.
	FilestoreDriverServer = "server"
	// FilestoreDriverS3 stores legal holds in a dedicated S3 bucket.
	FilestoreDriverS3 = "amazons3"
	// FilestoreDriverAzure stores legal holds in a dedicated Azure Blob container.
	FilestoreDriverAzure = "azureblob"
	// FilestoreDriverLocal stores legal holds in a directory on the server.
	FilestoreDriverLocal = "local"
)

type Configuration struct {
	TimeOfDay                     string
	EnableFilestoreConnectionTest bool
	AmazonS3BucketSettings        AmazonS3BucketSettings
}

// AmazonS3BucketSettings is the dedicated filestore section. The config key
// stays amazons3bucketsettings so existing installations keep their saved values.
// DriverName selects which fields are in use. Enable is the pre-driver flag:
// true meant "use the S3 settings".
type AmazonS3BucketSettings struct {
	Enable     bool
	DriverName string
	Settings   model.FileSettings
	Azure      AzureBlobSettings
	Directory  string
}

// AzureBlobSettings is the dedicated Azure Blob container for legal holds.
type AzureBlobSettings struct {
	StorageAccount             string
	AuthMode                   string
	AccessKey                  string
	Container                  string
	PathPrefix                 string
	Cloud                      string
	Endpoint                   string
	SSL                        bool
	RequestTimeoutMilliseconds int64
}

// EffectiveDriver resolves the store the admin selected. A config saved before
// DriverName existed used Enable alone: on meant S3, off meant the server store.
func (s AmazonS3BucketSettings) EffectiveDriver() string {
	switch s.DriverName {
	case FilestoreDriverServer, FilestoreDriverS3, FilestoreDriverAzure, FilestoreDriverLocal:
		return s.DriverName
	}
	if s.Enable {
		return FilestoreDriverS3
	}
	return FilestoreDriverServer
}

// Clone shallow copies the Configuration. Your implementation may require a deep copy if
// your Configuration has reference types.
func (c *Configuration) Clone() *Configuration {
	clone := *c
	return &clone
}
