package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_getConfiguration(t *testing.T) {
	p := &Plugin{}
	api := &plugintest.API{}
	p.SetAPI(api)

	c := p.getConfiguration()
	assert.NotNil(t, c)
}
