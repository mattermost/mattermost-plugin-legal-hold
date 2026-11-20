package azureblob

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	// DriverName is Mattermost FileSettings.DriverName for the native Azure
	// Blob backend. A blobfuse mount stays DriverName "local" and does not
	// come through this package.
	DriverName = "azureblob"

	AuthModeSharedKey         = "shared_key"
	AuthModeDefaultCredential = "default_credential"

	CloudCommercial = "commercial"
	CloudGovernment = "government"
	CloudCustom     = "custom"
)

// Settings is the subset of FileSettings the Azure backend needs.
// The plugin's compiled model is mattermost-server v6, which has no Azure
// fields, so these are recovered from the stored config JSON and from
// MM_FILESETTINGS_AZURE* environment variables instead of from GetConfig().
type Settings struct {
	StorageAccount             string
	AuthMode                   string
	AccessKey                  string
	Container                  string
	PathPrefix                 string
	Cloud                      string
	Endpoint                   string
	SSL                        *bool
	RequestTimeoutMilliseconds *int64
}

func (s Settings) ssl() bool {
	return s.SSL == nil || *s.SSL
}

func (s Settings) timeoutMilliseconds() int64 {
	if s.RequestTimeoutMilliseconds == nil || *s.RequestTimeoutMilliseconds <= 0 {
		return 30000
	}
	return *s.RequestTimeoutMilliseconds
}

func (s Settings) withDefaults() Settings {
	if s.AuthMode == "" {
		s.AuthMode = AuthModeSharedKey
	}
	if s.Cloud == "" {
		s.Cloud = CloudCommercial
	}
	return s
}

// CheckMandatory rejects a configuration that cannot authenticate or name a container.
func (s Settings) CheckMandatory() error {
	s = s.withDefaults()
	if s.StorageAccount == "" {
		return fmt.Errorf("missing azure storage account setting")
	}
	if s.Container == "" {
		return fmt.Errorf("missing azure container setting")
	}
	// Default credential reads identity from the environment (managed identity,
	// workload identity, service principal). An empty access key is expected there.
	if s.AuthMode == AuthModeSharedKey && s.AccessKey == "" {
		return fmt.Errorf("missing azure access key setting")
	}
	if s.AuthMode != AuthModeSharedKey && s.AuthMode != AuthModeDefaultCredential {
		return fmt.Errorf("unknown azure auth mode %q", s.AuthMode)
	}
	return nil
}

type storedConfig struct {
	FileSettings storedFileSettings `json:"FileSettings"`
}

type storedFileSettings struct {
	AzureStorageAccount             *string `json:"AzureStorageAccount"`
	AzureAuthMode                   *string `json:"AzureAuthMode"`
	AzureAccessKey                  *string `json:"AzureAccessKey"`
	AzureContainer                  *string `json:"AzureContainer"`
	AzurePathPrefix                 *string `json:"AzurePathPrefix"`
	AzureCloud                      *string `json:"AzureCloud"`
	AzureEndpoint                   *string `json:"AzureEndpoint"`
	AzureSSL                        *bool   `json:"AzureSSL"`
	AzureRequestTimeoutMilliseconds *int64  `json:"AzureRequestTimeoutMilliseconds"`
}

// SettingsFromConfigJSON reads Azure file settings out of a Mattermost config
// document (config.json or the configurations.value column). An empty payload
// returns zero settings so callers can still apply environment overrides.
func SettingsFromConfigJSON(raw []byte) (Settings, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return Settings{}, nil
	}
	var doc storedConfig
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Settings{}, fmt.Errorf("parse mattermost config: %w", err)
	}
	fs := doc.FileSettings
	return Settings{
		StorageAccount:             deref(fs.AzureStorageAccount),
		AuthMode:                   deref(fs.AzureAuthMode),
		AccessKey:                  deref(fs.AzureAccessKey),
		Container:                  deref(fs.AzureContainer),
		PathPrefix:                 deref(fs.AzurePathPrefix),
		Cloud:                      deref(fs.AzureCloud),
		Endpoint:                   deref(fs.AzureEndpoint),
		SSL:                        fs.AzureSSL,
		RequestTimeoutMilliseconds: fs.AzureRequestTimeoutMilliseconds,
	}, nil
}

// ApplyEnv overlays MM_FILESETTINGS_AZURE* the same way the server does.
// A variable that is present replaces the stored value. An unset variable
// leaves the stored value alone, so an empty environment does not wipe
// settings that came from config.json or the configurations table.
func ApplyEnv(s Settings, lookup func(string) (string, bool)) Settings {
	if lookup == nil {
		return s
	}
	setString := func(key string, dest *string) {
		if v, ok := lookup(key); ok {
			*dest = v
		}
	}
	setString("MM_FILESETTINGS_AZURESTORAGEACCOUNT", &s.StorageAccount)
	setString("MM_FILESETTINGS_AZUREAUTHMODE", &s.AuthMode)
	setString("MM_FILESETTINGS_AZUREACCESSKEY", &s.AccessKey)
	setString("MM_FILESETTINGS_AZURECONTAINER", &s.Container)
	setString("MM_FILESETTINGS_AZUREPATHPREFIX", &s.PathPrefix)
	setString("MM_FILESETTINGS_AZURECLOUD", &s.Cloud)
	setString("MM_FILESETTINGS_AZUREENDPOINT", &s.Endpoint)
	if v, ok := lookup("MM_FILESETTINGS_AZURESSL"); ok {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			s.SSL = &parsed
		}
	}
	if v, ok := lookup("MM_FILESETTINGS_AZUREREQUESTTIMEOUTMILLISECONDS"); ok {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			s.RequestTimeoutMilliseconds = &parsed
		}
	}
	return s
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

var azureStorageAccountName = regexp.MustCompile(`^[a-z0-9]{3,24}$`)

func validAccountName(name string) bool {
	return azureStorageAccountName.MatchString(name)
}

// ServiceURL is the blob endpoint the SDK signs against.
// Commercial and government are vhost-style. Custom is the admin-supplied
// URL unchanged (Azurite, a proxy, or a non-public cloud).
func ServiceURL(s Settings) (string, error) {
	s = s.withDefaults()
	scheme := "https"
	if !s.ssl() {
		scheme = "http"
	}
	switch s.Cloud {
	case CloudCommercial:
		if !validAccountName(s.StorageAccount) {
			return "", fmt.Errorf("invalid azure storage account name %q", s.StorageAccount)
		}
		return fmt.Sprintf("%s://%s.blob.core.windows.net/", scheme, s.StorageAccount), nil
	case CloudGovernment:
		if !validAccountName(s.StorageAccount) {
			return "", fmt.Errorf("invalid azure storage account name %q", s.StorageAccount)
		}
		return fmt.Sprintf("%s://%s.blob.core.usgovcloudapi.net/", scheme, s.StorageAccount), nil
	case CloudCustom:
		if s.Endpoint == "" {
			return "", fmt.Errorf("AzureCloud=custom requires AzureEndpoint to be set")
		}
		parsed, err := url.Parse(s.Endpoint)
		if err != nil {
			return "", fmt.Errorf("AzureEndpoint is not a valid URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return "", fmt.Errorf("AzureEndpoint must use http or https, got %q", parsed.Scheme)
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("AzureEndpoint must include a host, got %q", s.Endpoint)
		}
		return s.Endpoint, nil
	default:
		return "", fmt.Errorf("unknown AzureCloud value %q", s.Cloud)
	}
}
