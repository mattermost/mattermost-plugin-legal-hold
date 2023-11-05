package kvstore

import (
	"encoding/json"
	"fmt"
	"testing"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

func TestKVStore_CreateLegalHold(t *testing.T) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	kvstore := NewKVStore(client)

	// Test with a fresh legal hold
	lh1 := model.LegalHold{
		ID:   mattermostModel.NewId(),
		Name: "legal-hold-1",
	}

	api.On("KVSetWithOptions",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("model.PluginKVSetOptions"),
	).Run(func(args mock.Arguments) {
		marshalled := args.Get(1).([]uint8)
		api.On("KVGet", mock.AnythingOfType("string")).Return(marshalled, nil).Once()
	}).Return(true, nil).Once()

	lh2, err := kvstore.CreateLegalHold(lh1)

	require.NoError(t, err)
	assert.Equal(t, lh1.ID, lh2.ID)
	assert.Equal(t, lh1.Name, lh2.Name)

	// Test with a legal hold with duplicate ID
	lh3 := model.LegalHold{
		ID:   mattermostModel.NewId(),
		Name: "legal-hold-3",
	}

	api.On("KVSetWithOptions",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("model.PluginKVSetOptions"),
	).Return(false, nil).Once()

	_, err = kvstore.CreateLegalHold(lh3)
	require.Error(t, err)
}

func TestKVStore_GetLegalHoldById(t *testing.T) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	kvstore := NewKVStore(client)

	lh1 := model.LegalHold{
		ID:   mattermostModel.NewId(),
		Name: "legal-hold-1",
	}
	marshalled, err := json.Marshal(lh1)
	require.NoError(t, err)

	api.On("KVGet", fmt.Sprintf("%s%s", legalHoldPrefix, lh1.ID)).
		Return(marshalled, nil)

	// Test getting a valid legal hold
	lh2, err := kvstore.GetLegalHoldByID(lh1.ID)
	require.NoError(t, err)
	require.Equal(t, lh1, *lh2)

	api.On("KVGet", mock.AnythingOfType("string")).
		Return(nil, &mattermostModel.AppError{})

	// Test getting one by ID that does not exist
	lh3, err := kvstore.GetLegalHoldByID("doesnotexist")
	require.Error(t, err)
	require.Nil(t, lh3)
}

func TestKVStore_GetAllLegalHolds(t *testing.T) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	kvstore := NewKVStore(client)

	lh1 := model.LegalHold{
		ID:   mattermostModel.NewId(),
		Name: "legal-hold-1",
	}

	lh2 := model.LegalHold{
		ID:   mattermostModel.NewId(),
		Name: "legal-hold-2",
	}

	lhs := []model.LegalHold{lh1, lh2}

	marshalled1, err := json.Marshal(lh1)
	require.NoError(t, err)

	marshalled2, err := json.Marshal(lh2)
	require.NoError(t, err)

	api.On("KVList", mock.AnythingOfType("int"), mock.AnythingOfType("int")).
		Return([]string{
			fmt.Sprintf("%s%s", legalHoldPrefix, lh1.ID),
			fmt.Sprintf("%s%s", legalHoldPrefix, lh2.ID),
		}, nil).
		Once()

	api.On("KVGet", fmt.Sprintf("%s%s", legalHoldPrefix, lh1.ID)).Return(marshalled1, nil)
	api.On("KVGet", fmt.Sprintf("%s%s", legalHoldPrefix, lh2.ID)).Return(marshalled2, nil)

	// Test with some data
	result, err := kvstore.GetAllLegalHolds()
	require.NoError(t, err)
	require.Equal(t, lhs, result)

	// Test with no data
	api.On("KVList", mock.AnythingOfType("int"), mock.AnythingOfType("int")).
		Return([]string{}, nil).
		Once()

	result, err = kvstore.GetAllLegalHolds()
	require.NoError(t, err)
	require.Len(t, result, 0)
}
