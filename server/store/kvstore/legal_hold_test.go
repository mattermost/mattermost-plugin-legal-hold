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
		ID:          mattermostModel.NewId(),
		Name:        "legal-hold-1",
		DisplayName: "Legal Hold 1",
		UserIDs:     []string{mattermostModel.NewId()},
		StartsAt:    mattermostModel.GetMillis(),
	}

	api.On("KVSetWithOptions",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("model.PluginKVSetOptions"),
	).Run(func(args mock.Arguments) {
		marshaled := args.Get(1).([]uint8)
		api.On("KVGet", mock.AnythingOfType("string")).Return(marshaled, nil).Once()
	}).Return(true, nil).Once()

	lh2, err := kvstore.CreateLegalHold(lh1)

	require.NoError(t, err)
	assert.Equal(t, lh1.ID, lh2.ID)
	assert.Equal(t, lh1.Name, lh2.Name)
	assert.Equal(t, lh1.UserIDs, lh2.UserIDs)
	assert.Equal(t, lh1.StartsAt, lh2.StartsAt)
	assert.Equal(t, lh1.EndsAt, lh2.EndsAt)
	assert.Equal(t, lh1.DisplayName, lh2.DisplayName)
	assert.Equal(t, lh1.LastExecutionEndedAt, lh2.LastExecutionEndedAt)
	assert.Equal(t, lh1.ExecutionLength, lh2.ExecutionLength)
	assert.Equal(t, int64(0), lh1.CreateAt)
	assert.Equal(t, int64(0), lh1.UpdateAt)
	assert.NotEqual(t, int64(0), lh2.CreateAt)
	assert.NotEqual(t, int64(0), lh2.UpdateAt)
	assert.Equal(t, lh2.CreateAt, lh2.UpdateAt)

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
	marshaled, err := json.Marshal(lh1)
	require.NoError(t, err)

	api.On("KVGet", fmt.Sprintf("%s%s", legalHoldPrefix, lh1.ID)).
		Return(marshaled, nil)
	api.On("KVGet", getLegalHoldLockKey(lh1.ID)).
		Return(nil, nil)

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

	marshaled1, err := json.Marshal(lh1)
	require.NoError(t, err)

	marshaled2, err := json.Marshal(lh2)
	require.NoError(t, err)

	api.On("KVList", mock.AnythingOfType("int"), mock.AnythingOfType("int")).
		Return([]string{
			fmt.Sprintf("%s%s", legalHoldPrefix, lh1.ID),
			fmt.Sprintf("%s%s", legalHoldPrefix, lh2.ID),
		}, nil).
		Once()

	api.On("KVGet", fmt.Sprintf("%s%s", legalHoldPrefix, lh1.ID)).Return(marshaled1, nil)
	api.On("KVGet", getLegalHoldLockKey(lh1.ID)).Return(nil, nil)
	api.On("KVGet", fmt.Sprintf("%s%s", legalHoldPrefix, lh2.ID)).Return(marshaled2, nil)
	api.On("KVGet", getLegalHoldLockKey(lh2.ID)).Return(nil, nil)

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

func TestKVStore_UpdateLegalHold(t *testing.T) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	kvstore := NewKVStore(client)

	// Original legal hold
	lh1 := model.LegalHold{
		ID:          mattermostModel.NewId(),
		Name:        "legal-hold-1",
		DisplayName: "Legal Hold 1",
		UserIDs:     []string{mattermostModel.NewId()},
		StartsAt:    mattermostModel.GetMillis(),
	}

	// Update legal hold
	lh2 := model.LegalHold{
		ID:          lh1.ID,
		Name:        "legal-hold-1",
		DisplayName: "Legal Hold 2",
		UserIDs:     []string{mattermostModel.NewId()},
		StartsAt:    mattermostModel.GetMillis(),
	}

	api.On("KVSetWithOptions",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("model.PluginKVSetOptions"),
	).Run(func(args mock.Arguments) {
		marshaled := args.Get(1).([]uint8)
		api.On("KVGet", mock.AnythingOfType("string")).Return(marshaled, nil).Once()
	}).Return(true, nil).Once()

	// Test updating a legal hold
	lh3, err := kvstore.UpdateLegalHold(lh2, lh1)
	require.NoError(t, err)
	assert.Equal(t, lh3.ID, lh2.ID)
	assert.Equal(t, lh3.Name, lh2.Name)
	assert.Equal(t, lh3.UserIDs, lh2.UserIDs)
	assert.Equal(t, lh3.StartsAt, lh2.StartsAt)
	assert.Equal(t, lh3.EndsAt, lh2.EndsAt)
	assert.Equal(t, lh3.DisplayName, lh2.DisplayName)
	assert.Equal(t, lh3.LastExecutionEndedAt, lh2.LastExecutionEndedAt)
	assert.Equal(t, lh3.ExecutionLength, lh2.ExecutionLength)
	assert.NotEqual(t, lh3.UpdateAt, lh2.UpdateAt)
	assert.Equal(t, lh3.CreateAt, lh2.UpdateAt)

	// Test updating a legal hold that does not exist
	lh4 := model.LegalHold{
		ID:          "doesnotexist",
		Name:        "legal-hold-4",
		DisplayName: "Legal Hold 4",
		UserIDs:     []string{mattermostModel.NewId()},
		StartsAt:    mattermostModel.GetMillis(),
	}
	_, err = kvstore.UpdateLegalHold(lh4, *lh3)
	require.Error(t, err)
}

func TestKVStore_DeleteLegalHold(t *testing.T) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	kvstore := NewKVStore(client)

	// Test deleting a legal hold that exists
	lhID := mattermostModel.NewId()
	api.On("KVSetWithOptions",
		fmt.Sprintf("%s%s", legalHoldPrefix, lhID),
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("model.PluginKVSetOptions"),
	).Return(true, nil).Once()

	err := kvstore.DeleteLegalHold(lhID)
	require.NoError(t, err)

	// Test deleting a legal hold that doesn't exist
	api.On("KVSetWithOptions",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("model.PluginKVSetOptions"),
	).Return(false, &mattermostModel.AppError{})

	err = kvstore.DeleteLegalHold("does-not-exist")
	require.Error(t, err)
}

func TestKVStore_LockLegalHold(t *testing.T) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	kvstore := NewKVStore(client)

	t.Run("lock ok", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil).Once()
		api.On("KVSetWithOptions",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("[]uint8"),
			mock.AnythingOfType("model.PluginKVSetOptions"),
		).Run(func(args mock.Arguments) {
			marshaled := args.Get(1).([]uint8)
			var checkLH bool
			require.NoError(t, json.Unmarshal(marshaled, &checkLH))
			require.True(t, checkLH)
			api.On("KVGet", mock.AnythingOfType("string")).Return(marshaled, nil).Once()
		}).Return(true, nil).Once()

		err := kvstore.LockLegalHold("legal_hold_id")
		require.NoError(t, err)
	})

	t.Run("can't lock twice", func(t *testing.T) {
		// Fake existing key
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte{0x74, 0x72, 0x75, 0x65}, nil).Once()
		err := kvstore.LockLegalHold("legal_hold_id")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAlreadyLocked)
	})
}

func TestKVStore_UnlockLegalHold(t *testing.T) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	kvstore := NewKVStore(client)

	t.Run("unlock ok", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte{0x74, 0x72, 0x75, 0x65}, nil).Once()
		api.On("KVSetWithOptions",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("[]uint8"),
			mock.AnythingOfType("model.PluginKVSetOptions"),
		).Run(func(args mock.Arguments) {
			marshaled := args.Get(1)
			require.Empty(t, marshaled)
		}).Return(true, nil).Once()

		err := kvstore.UnlockLegalHold("legal_hold_id")
		require.NoError(t, err)
	})

	t.Run("can't unlock twice", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil).Once()
		err := kvstore.UnlockLegalHold("legal_hold_id")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAlreadyUnlocked)
	})
}

func TestKVStore_IsLockedLegalHold(t *testing.T) {
	api := &plugintest.API{}
	driver := &plugintest.Driver{}
	client := pluginapi.NewClient(api, driver)

	kvstore := NewKVStore(client)

	t.Run("locked", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return([]byte{0x74, 0x72, 0x75, 0x65}, nil).Once()
		locked, err := kvstore.IsLockedLegalHold("legal_hold_id")
		require.NoError(t, err)
		require.True(t, locked)
	})

	t.Run("unlocked", func(t *testing.T) {
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil).Once()
		locked, err := kvstore.IsLockedLegalHold("legal_hold_id")
		require.NoError(t, err)
		require.False(t, locked)
	})
}
