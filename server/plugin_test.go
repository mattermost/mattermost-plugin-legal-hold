package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_getConfiguration(t *testing.T) {
	p := &Plugin{}
	api := &plugintest.API{}
	p.SetAPI(api)

	c := p.getConfiguration()
	assert.NotNil(t, c)
}

func TestFixedFileSettingsToFileBackendSettings(t *testing.T) {
	f := model.FileSettings{
		DriverName:                         model.NewString("amazons3"),
		AmazonS3Bucket:                     model.NewString("bucket"),
		AmazonS3AccessKeyId:                model.NewString("access_key_id"),
		AmazonS3SecretAccessKey:            model.NewString("secret_access_key"),
		AmazonS3RequestTimeoutMilliseconds: model.NewInt64(5000),
		AmazonS3Endpoint:                   model.NewString("localhost:8080"),
		AmazonS3Region:                     model.NewString("us-east-1"),
		AmazonS3SSL:                        model.NewBool(false),
		AmazonS3SSE:                        model.NewBool(false),
	}

	t.Run("with compliance enabled", func(t *testing.T) {
		result := FixedFileSettingsToFileBackendSettings(f, true, false)
		assert.True(t, result.AmazonS3SSE)
	})

	t.Run("with compliance disabled", func(t *testing.T) {
		result := FixedFileSettingsToFileBackendSettings(f, false, false)
		assert.False(t, result.AmazonS3SSE)
	})
}
