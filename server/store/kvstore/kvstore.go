package kvstore

import "github.com/mattermost/mattermost-plugin-legal-hold/server/model"

type KVStore interface {
	CreateLegalHold(lh model.LegalHold) (*model.LegalHold, error)
	GetAllLegalHolds() ([]model.LegalHold, error)
	GetLegalHoldByID(id string) (*model.LegalHold, error)
	UpdateLegalHold(lh, oldValue model.LegalHold) (*model.LegalHold, error)
	DeleteLegalHold(id string) error
	LockLegalHold(id string) error
	UnlockLegalHold(id string) error
	IsLockedLegalHold(id string) (bool, error)
}
