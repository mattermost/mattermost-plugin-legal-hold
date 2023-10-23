package kvstore

import (
	"fmt"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

const (
	legalHoldPrefix = "kvstore_legal_hold_"
)

type KVStore struct {
	client *pluginapi.Client
}

func NewKVStore(client *pluginapi.Client) KVStore {
	return KVStore{
		client: client,
	}
}

func (kvs *KVStore) CreateLegalHold(lh model.LegalHold) (*model.LegalHold, error) {
	if err := lh.IsValidForCreate(); err != nil {
		return nil, errors.Wrap(err, "LegalHold is not valid for create")
	}

	lh.CreateAt = mattermostModel.GetMillis()
	lh.UpdateAt = lh.CreateAt

	key := fmt.Sprintf("%s%s", legalHoldPrefix, lh.ID)

	saved, err := kvs.client.KV.Set(key, lh, pluginapi.SetAtomic(nil))
	if !saved && err != nil {
		return nil, errors.Wrap(err, "database error occurred creating legal hold")
	} else if !saved && err == nil {
		return nil, errors.New("could not create legal hold as a legal hold with that ID already eixsts.")
	}

	var savedLegalHold model.LegalHold
	err = kvs.client.KV.Get(key, &savedLegalHold)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get legal hold after creating it")
	}

	return &savedLegalHold, nil
}

func (kvs *KVStore) GetAllLegalHolds() ([]model.LegalHold, error) {
	keys, err := kvs.client.KV.ListKeys(
		0, 1000000000,
		pluginapi.WithPrefix(legalHoldPrefix))

	if err != nil {
		return nil, errors.Wrap(err, "could not get all legal holds")
	}

	var legalHolds []model.LegalHold
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

func (kvs *KVStore) UpdateLegalHold(lh, oldValue model.LegalHold) (*model.LegalHold, error) {
	// FIXME: Should validation function be different for update?
	if err := lh.IsValidForCreate(); err != nil {
		return nil, errors.Wrap(err, "LegalHold is not valid for create")
	}

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
		return nil, errors.Wrap(err, "failed to get legal hold after creating it")
	}

	return &savedLegalHold, nil
}
