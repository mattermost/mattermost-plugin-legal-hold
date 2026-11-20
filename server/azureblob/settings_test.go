package azureblob

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSettingsFromConfigJSONAndEnv(t *testing.T) {
	raw := []byte(`{
		"FileSettings": {
			"DriverName": "azureblob",
			"AzureStorageAccount": "fromjson",
			"AzureContainer": "mattermost",
			"AzureAccessKey": "json-key",
			"AzureAuthMode": "shared_key",
			"AzureCloud": "government"
		}
	}`)
	got, err := SettingsFromConfigJSON(raw)
	require.NoError(t, err)
	require.Equal(t, "fromjson", got.StorageAccount)
	require.Equal(t, "government", got.Cloud)

	env := map[string]string{
		"MM_FILESETTINGS_AZURESTORAGEACCOUNT": "fromenv",
		"MM_FILESETTINGS_AZURESSL":            "false",
	}
	got = ApplyEnv(got, func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	})
	require.Equal(t, "fromenv", got.StorageAccount)
	require.Equal(t, "mattermost", got.Container)
	require.Equal(t, "json-key", got.AccessKey)
	require.NotNil(t, got.SSL)
	require.False(t, *got.SSL)

	require.NoError(t, got.CheckMandatory())
	url, err := ServiceURL(got)
	require.NoError(t, err)
	require.Equal(t, "http://fromenv.blob.core.usgovcloudapi.net/", url)
}

func TestServiceURLCustomRequiresEndpoint(t *testing.T) {
	_, err := ServiceURL(Settings{
		StorageAccount: "acct",
		Cloud:          CloudCustom,
	})
	require.Error(t, err)

	url, err := ServiceURL(Settings{
		StorageAccount: "devstoreaccount1",
		Cloud:          CloudCustom,
		Endpoint:       "http://127.0.0.1:10000/devstoreaccount1",
	})
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:10000/devstoreaccount1", url)
}

func TestCheckMandatoryDefaultCredentialSkipsKey(t *testing.T) {
	err := Settings{
		StorageAccount: "acctname",
		Container:      "files",
		AuthMode:       AuthModeDefaultCredential,
	}.CheckMandatory()
	require.NoError(t, err)

	err = Settings{
		StorageAccount: "acctname",
		Container:      "files",
		AuthMode:       AuthModeSharedKey,
	}.CheckMandatory()
	require.Error(t, err)
}

func TestStripPrefixUsesCleanedPrefix(t *testing.T) {
	b := &Backend{pathPrefix: cleanPathPrefix("./holds")}
	written, err := b.prefix("legalhold/index.json")
	require.NoError(t, err)
	require.Equal(t, "holds/legalhold/index.json", written)
	require.Equal(t, "legalhold/index.json", b.stripPrefix(written))
}

func TestDotPrefixKeepsNestedNamesDistinct(t *testing.T) {
	b := &Backend{pathPrefix: cleanPathPrefix("./")}
	require.Empty(t, b.pathPrefix)
	first, err := b.prefix("legalhold/a/index.json")
	require.NoError(t, err)
	second, err := b.prefix("legalhold/b/index.json")
	require.NoError(t, err)
	require.Equal(t, "legalhold/a/index.json", first)
	require.Equal(t, "legalhold/b/index.json", second)
	require.NotEqual(t, first, second)
	require.Equal(t, first, b.stripPrefix(first))
	require.Equal(t, second, b.stripPrefix(second))
	require.Equal(t, "holds", cleanPathPrefix("./holds"))
}

func TestContainerRootPrefixIsEmpty(t *testing.T) {
	b := &Backend{}
	root, err := b.prefix("")
	require.NoError(t, err)
	require.Empty(t, root)
	dot, err := b.prefix(".")
	require.NoError(t, err)
	require.Empty(t, dot)
	key, err := b.prefix("legalhold/index.json")
	require.NoError(t, err)
	require.Equal(t, "legalhold/index.json", key)
}

func TestPrefixRejectsParentSegment(t *testing.T) {
	b := &Backend{pathPrefix: "tenant/holds"}
	_, err := b.prefix("..")
	require.Error(t, err)
	_, err = b.prefix("nested/..")
	require.Error(t, err)
	key, err := b.prefix("foo/../../secret")
	require.NoError(t, err)
	require.Equal(t, "tenant/holds/secret", key)
}

func TestSettingsFromConfigJSONEmpty(t *testing.T) {
	got, err := SettingsFromConfigJSON(nil)
	require.NoError(t, err)
	require.Empty(t, got.StorageAccount)
}
