package kvstore

import (
	"fmt"

	"github.com/pkg/errors"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

const (
	legalHoldPrefix = "kvstore_legal_hold_"
)

type Impl struct {
	client *pluginapi.Client
}

func NewKVStore(client *pluginapi.Client) KVStore {
	return Impl{
		client: client,
	}
}

// unorderedEqualSet compares two slices of user IDs regardless of order
func unorderedEqualSet[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	setA := make(map[T]bool, len(a))
	for _, v := range a {
		setA[v] = true
	}

	for _, v := range b {
		if !setA[v] {
			return false
		}
	}

	return true
}

func (kvs Impl) CreateLegalHold(lh model.LegalHold) (*model.LegalHold, error) {

	// Check for duplicates by name, dates, and participants
	existingLegalHolds, err := kvs.GetAllLegalHolds()
	if err != nil {
		return nil, errors.Wrap(err, "failed to check for existing legal holds")
	}

	for _, existing := range existingLegalHolds {
		if existing.Name == lh.Name {
			return nil, errors.New("could not create legal hold as a legal hold with that name already exists")
		}

		// Check for functional duplicates (same participants, dates, and channel settings)
		if existing.StartsAt == lh.StartsAt &&
			existing.EndsAt == lh.EndsAt &&
			existing.IncludePublicChannels == lh.IncludePublicChannels &&
			unorderedEqualSet(existing.UserIDs, lh.UserIDs) {
			return nil, errors.New("could not create legal hold as a legal hold with the same participants, dates, and settings already exists")
		}
	}

	lh.CreateAt = mattermostModel.GetMillis()
	lh.UpdateAt = lh.CreateAt
	lh.Secret = mattermostModel.NewId()

	key := fmt.Sprintf("%s%s", legalHoldPrefix, lh.ID)

	saved, err := kvs.client.KV.Set(key, lh, pluginapi.SetAtomic(nil))
	if !saved && err != nil {
		return nil, errors.Wrap(err, "database error occurred creating legal hold")
	} else if !saved && err == nil {
		return nil, errors.New("could not create legal hold as a legal hold with that ID already exists")
	}

	var savedLegalHold model.LegalHold
	err = kvs.client.KV.Get(key, &savedLegalHold)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get legal hold after creating it")
	}

	return &savedLegalHold, nil
}

func (kvs Impl) GetAllLegalHolds() ([]model.LegalHold, error) {
	keys, err := kvs.client.KV.ListKeys(
		0, 1000000000,
		pluginapi.WithPrefix(legalHoldPrefix))

	if err != nil {
		return nil, errors.Wrap(err, "could not get all legal holds")
	}

	var legalHolds = make([]model.LegalHold, 0)
	for _, key := range keys {
		var legalHold model.LegalHold
		err = kvs.client.KV.Get(key, &legalHold)
		if err != nil {
			return nil, errors.Wrap(err, "could not get all legal holds")
		}
		legalHolds = append(legalHolds, legalHold)
	}

	return legalHolds, nil
}

func (kvs Impl) GetLegalHoldByID(id string) (*model.LegalHold, error) {
	key := fmt.Sprintf("%s%s", legalHoldPrefix, id)

	var legalHold model.LegalHold
	if err := kvs.client.KV.Get(key, &legalHold); err != nil {
		return nil, errors.Wrap(err, "could not get legal hold by id")
	}

	return &legalHold, nil
}

func (kvs Impl) UpdateLegalHold(lh, oldValue model.LegalHold) (*model.LegalHold, error) {
	lh.UpdateAt = mattermostModel.GetMillis()

	key := fmt.Sprintf("%s%s", legalHoldPrefix, lh.ID)

	saved, err := kvs.client.KV.Set(key, lh, pluginapi.SetAtomic(oldValue))
	if !saved && err != nil {
		return nil, errors.Wrap(err, "database error occurred updating legal hold")
	} else if !saved && err == nil {
		return nil, errors.New("could not update legal hold as it has already been updated by someone else")
	}

	var savedLegalHold model.LegalHold
	err = kvs.client.KV.Get(key, &savedLegalHold)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get legal hold after updating it")
	}

	return &savedLegalHold, nil
}

func (kvs Impl) DeleteLegalHold(id string) error {
	key := fmt.Sprintf("%s%s", legalHoldPrefix, id)

	err := kvs.client.KV.Delete(key)
	return err
}
