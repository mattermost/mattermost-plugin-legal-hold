package kvstore

import (
	"fmt"
	"time"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

const (
	legalHoldPrefix = "kvstore_legal_hold_"
	lockExpiry      = 24 * time.Hour
)

var ErrAlreadyLocked = errors.New("legal hold is already locked")
var ErrAlreadyUnlocked = errors.New("legal hold is already unlocked")

func getLegalHoldLockKey(legalHoldID string) string {
	return legalHoldPrefix + "_lock_" + legalHoldID
}

type Impl struct {
	client *pluginapi.Client
}

func NewKVStore(client *pluginapi.Client) KVStore {
	return Impl{
		client: client,
	}
}

func (kvs Impl) CreateLegalHold(lh model.LegalHold) (*model.LegalHold, error) {
	if err := lh.IsValidForCreate(); err != nil {
		return nil, errors.Wrap(err, "LegalHold is not valid for create")
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
		isLocked, err := kvs.IsLockedLegalHold(legalHold.ID)
		if err != nil {
			return nil, errors.Wrap(err, "could not get all legal holds")
		}
		legalHold.Locked = isLocked
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

	isLocked, err := kvs.IsLockedLegalHold(legalHold.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get legal hold by id")
	}
	legalHold.Locked = isLocked

	return &legalHold, nil
}

func (kvs Impl) UpdateLegalHold(lh, oldValue model.LegalHold) (*model.LegalHold, error) {
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

func (kvs Impl) DeleteLegalHold(id string) error {
	key := fmt.Sprintf("%s%s", legalHoldPrefix, id)

	err := kvs.client.KV.Delete(key)
	return err
}

func (kvs Impl) LockLegalHold(legalHoldID string) error {
	locked, err := kvs.IsLockedLegalHold(legalHoldID)
	if err != nil {
		return fmt.Errorf("could not lock legal hold: %w", err)
	}
	if locked {
		return ErrAlreadyLocked
	}

	stored, err := kvs.client.KV.Set(
		getLegalHoldLockKey(legalHoldID),
		true,
		pluginapi.SetAtomic(nil),
		pluginapi.SetExpiry(lockExpiry),
	)
	if err != nil {
		return fmt.Errorf("could not lock legal hold: %w", err)
	}

	if !stored {
		return fmt.Errorf("could not store legal hold lock")
	}

	return nil
}

func (kvs Impl) UnlockLegalHold(legalHoldID string) error {
	locked, err := kvs.IsLockedLegalHold(legalHoldID)
	if err != nil {
		return fmt.Errorf("could not unlock legal hold: %w", err)
	}
	if !locked {
		return ErrAlreadyUnlocked
	}

	err = kvs.client.KV.Delete(getLegalHoldLockKey(legalHoldID))
	if err != nil {
		return fmt.Errorf("could not unlock legal hold: %w", err)
	}

	return nil
}

func (kvs Impl) IsLockedLegalHold(legalHoldID string) (bool, error) {
	var locked *bool
	err := kvs.client.KV.Get(getLegalHoldLockKey(legalHoldID), &locked)
	if err != nil {
		return false, fmt.Errorf("could not get legal hold lock status: %w", err)
	}

	return locked != nil, nil
}
