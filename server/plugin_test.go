package main

import (
	"testing"

	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_getConfiguration(t *testing.T) {
	p := &Plugin{}
	api := &plugintest.API{}
	p.SetAPI(api)

	c := p.getConfiguration()
	assert.NotNil(t, c)
}

func TestFixedFileSettingsToFileBackendSettings_LocalDriver(t *testing.T) {
	fileSettings := mattermostModel.FileSettings{
		DriverName: mattermostModel.NewString(mattermostModel.ImageDriverLocal),
		Directory:  mattermostModel.NewString("/data/files"),
	}

	result := FixedFileSettingsToFileBackendSettings(fileSettings)

	assert.Equal(t, mattermostModel.ImageDriverLocal, result.DriverName)
	assert.Equal(t, "/data/files", result.Directory)
	// S3 fields should be empty/default for local driver
	assert.Empty(t, result.AmazonS3Bucket)
}

func TestFixedFileSettingsToFileBackendSettings_S3WithAllFieldsSet(t *testing.T) {
	fileSettings := mattermostModel.FileSettings{
		DriverName:                         mattermostModel.NewString(mattermostModel.ImageDriverS3),
		AmazonS3AccessKeyId:                mattermostModel.NewString("access-key"),
		AmazonS3SecretAccessKey:            mattermostModel.NewString("secret-key"),
		AmazonS3Bucket:                     mattermostModel.NewString("my-bucket"),
		AmazonS3PathPrefix:                 mattermostModel.NewString("legal-hold"),
		AmazonS3Region:                     mattermostModel.NewString("us-east-1"),
		AmazonS3Endpoint:                   mattermostModel.NewString("s3.amazonaws.com"),
		AmazonS3SSL:                        mattermostModel.NewBool(true),
		AmazonS3SignV2:                     mattermostModel.NewBool(false),
		AmazonS3SSE:                        mattermostModel.NewBool(true),
		AmazonS3Trace:                      mattermostModel.NewBool(false),
		AmazonS3RequestTimeoutMilliseconds: mattermostModel.NewInt64(30000),
	}

	result := FixedFileSettingsToFileBackendSettings(fileSettings)

	assert.Equal(t, mattermostModel.ImageDriverS3, result.DriverName)
	assert.Equal(t, "access-key", result.AmazonS3AccessKeyId)
	assert.Equal(t, "secret-key", result.AmazonS3SecretAccessKey)
	assert.Equal(t, "my-bucket", result.AmazonS3Bucket)
	assert.Equal(t, "legal-hold", result.AmazonS3PathPrefix)
	assert.Equal(t, "us-east-1", result.AmazonS3Region)
	assert.Equal(t, "s3.amazonaws.com", result.AmazonS3Endpoint)
	assert.True(t, result.AmazonS3SSL)
	assert.False(t, result.AmazonS3SignV2)
	assert.True(t, result.AmazonS3SSE)
	assert.False(t, result.AmazonS3Trace)
	assert.Equal(t, int64(30000), result.AmazonS3RequestTimeoutMilliseconds)
	assert.False(t, result.SkipVerify)
}

func TestFixedFileSettingsToFileBackendSettings_S3WithNilBooleanFields(t *testing.T) {
	// This test verifies the nil-safe handling of boolean pointer fields.
	// When nil, they should default to safe values:
	// - AmazonS3SSL: true (secure by default)
	// - AmazonS3SignV2: false
	// - AmazonS3SSE: false
	// - AmazonS3Trace: false

	fileSettings := mattermostModel.FileSettings{
		DriverName:                         mattermostModel.NewString(mattermostModel.ImageDriverS3),
		AmazonS3AccessKeyId:                mattermostModel.NewString("access-key"),
		AmazonS3SecretAccessKey:            mattermostModel.NewString("secret-key"),
		AmazonS3Bucket:                     mattermostModel.NewString("my-bucket"),
		AmazonS3PathPrefix:                 mattermostModel.NewString(""),
		AmazonS3Region:                     mattermostModel.NewString("us-east-1"),
		AmazonS3Endpoint:                   mattermostModel.NewString(""),
		AmazonS3SSL:                        nil, // Should default to true
		AmazonS3SignV2:                     nil, // Should default to false
		AmazonS3SSE:                        nil, // Should default to false
		AmazonS3Trace:                      nil, // Should default to false
		AmazonS3RequestTimeoutMilliseconds: mattermostModel.NewInt64(30000),
	}

	result := FixedFileSettingsToFileBackendSettings(fileSettings)

	require.Equal(t, mattermostModel.ImageDriverS3, result.DriverName)
	assert.True(t, result.AmazonS3SSL, "AmazonS3SSL should default to true when nil")
	assert.False(t, result.AmazonS3SignV2, "AmazonS3SignV2 should default to false when nil")
	assert.False(t, result.AmazonS3SSE, "AmazonS3SSE should default to false when nil")
	assert.False(t, result.AmazonS3Trace, "AmazonS3Trace should default to false when nil")
}

func TestFixedFileSettingsToFileBackendSettings_S3WithNonDefaultBooleanValues(t *testing.T) {
	// This test verifies that when boolean fields are explicitly set to non-default values,
	// they are respected.

	fileSettings := mattermostModel.FileSettings{
		DriverName:                         mattermostModel.NewString(mattermostModel.ImageDriverS3),
		AmazonS3AccessKeyId:                mattermostModel.NewString("access-key"),
		AmazonS3SecretAccessKey:            mattermostModel.NewString("secret-key"),
		AmazonS3Bucket:                     mattermostModel.NewString("my-bucket"),
		AmazonS3PathPrefix:                 mattermostModel.NewString(""),
		AmazonS3Region:                     mattermostModel.NewString("us-east-1"),
		AmazonS3Endpoint:                   mattermostModel.NewString(""),
		AmazonS3SSL:                        mattermostModel.NewBool(false), // Explicitly set to non-default
		AmazonS3SignV2:                     mattermostModel.NewBool(true),  // Explicitly set to non-default
		AmazonS3SSE:                        mattermostModel.NewBool(true),  // Explicitly set to non-default
		AmazonS3Trace:                      mattermostModel.NewBool(true),  // Explicitly set to non-default
		AmazonS3RequestTimeoutMilliseconds: mattermostModel.NewInt64(30000),
	}

	result := FixedFileSettingsToFileBackendSettings(fileSettings)

	require.Equal(t, mattermostModel.ImageDriverS3, result.DriverName)
	assert.False(t, result.AmazonS3SSL, "AmazonS3SSL should be false when explicitly set")
	assert.True(t, result.AmazonS3SignV2, "AmazonS3SignV2 should be true when explicitly set")
	assert.True(t, result.AmazonS3SSE, "AmazonS3SSE should be true when explicitly set")
	assert.True(t, result.AmazonS3Trace, "AmazonS3Trace should be true when explicitly set")
}

func TestFixedFileSettingsToFileBackendSettings_NilPointerSafety(t *testing.T) {
	// This test verifies that the function doesn't panic when pointer fields are nil.
	// The original bug was direct dereferencing without checking for nil, causing:
	// "panic: runtime error: invalid memory address or nil pointer dereference"
	// SetDefaults(true) inside the function should initialize all nil fields safely.

	t.Run("S3WithNilStringFields", func(t *testing.T) {
		fileSettings := mattermostModel.FileSettings{
			DriverName:          mattermostModel.NewString(mattermostModel.ImageDriverS3),
			AmazonS3AccessKeyId: mattermostModel.NewString("access-key"),
			// All other string fields intentionally left as nil
			AmazonS3SecretAccessKey:            nil,
			AmazonS3Bucket:                     nil,
			AmazonS3PathPrefix:                 nil,
			AmazonS3Region:                     nil,
			AmazonS3Endpoint:                   nil,
			AmazonS3RequestTimeoutMilliseconds: nil,
		}

		// This should not panic
		require.NotPanics(t, func() {
			result := FixedFileSettingsToFileBackendSettings(fileSettings)

			// Verify the function completed successfully and returned sensible values
			assert.Equal(t, mattermostModel.ImageDriverS3, result.DriverName)
			assert.Equal(t, "access-key", result.AmazonS3AccessKeyId)
			// SetDefaults should have populated these with defaults
			assert.NotNil(t, result.AmazonS3Bucket)
			assert.NotNil(t, result.AmazonS3Region)
			assert.True(t, result.AmazonS3SSL, "AmazonS3SSL should default to true")
		})
	})

	t.Run("CompletelyEmptyFileSettings", func(t *testing.T) {
		// All fields are nil - this is the worst-case scenario
		fileSettings := mattermostModel.FileSettings{}

		// This should not panic - SetDefaults should initialize everything
		require.NotPanics(t, func() {
			result := FixedFileSettingsToFileBackendSettings(fileSettings)
			// After SetDefaults, DriverName should be initialized
			assert.NotEmpty(t, result.DriverName)
		})
	})

	t.Run("LocalDriverWithNilDirectory", func(t *testing.T) {
		fileSettings := mattermostModel.FileSettings{
			DriverName: mattermostModel.NewString(mattermostModel.ImageDriverLocal),
			Directory:  nil, // Intentionally nil
		}

		// Should not panic
		require.NotPanics(t, func() {
			result := FixedFileSettingsToFileBackendSettings(fileSettings)
			assert.Equal(t, mattermostModel.ImageDriverLocal, result.DriverName)
			// SetDefaults should have initialized Directory with a default value
			assert.NotEmpty(t, result.Directory)
		})
	})

	t.Run("S3WithNilDriverName", func(t *testing.T) {
		fileSettings := mattermostModel.FileSettings{
			DriverName:          nil, // Intentionally nil
			AmazonS3Bucket:      mattermostModel.NewString("my-bucket"),
			AmazonS3AccessKeyId: mattermostModel.NewString("access-key"),
		}

		// Should not panic even with nil DriverName
		require.NotPanics(t, func() {
			result := FixedFileSettingsToFileBackendSettings(fileSettings)
			// SetDefaults should have initialized DriverName with a default value
			assert.NotEmpty(t, result.DriverName)
		})
	})

	t.Run("S3WithNilTimeout", func(t *testing.T) {
		fileSettings := mattermostModel.FileSettings{
			DriverName:                         mattermostModel.NewString(mattermostModel.ImageDriverS3),
			AmazonS3Bucket:                     mattermostModel.NewString("my-bucket"),
			AmazonS3RequestTimeoutMilliseconds: nil, // Intentionally nil
		}

		// Should not panic with nil timeout
		require.NotPanics(t, func() {
			result := FixedFileSettingsToFileBackendSettings(fileSettings)
			assert.Equal(t, mattermostModel.ImageDriverS3, result.DriverName)
			// SetDefaults should have initialized timeout with a default value
			assert.NotZero(t, result.AmazonS3RequestTimeoutMilliseconds)
		})
	})
}
