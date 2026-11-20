package sqlstore

import (
	"database/sql"

	"github.com/mattermost/mattermost-server/v6/model"
)

// ActiveConfiguration returns the JSON document from the active row in
// Mattermost's configurations table. Config that lives only in config.json
// or only in environment variables is not in this table; callers overlay those.
func (ss *SQLStore) ActiveConfiguration() ([]byte, error) {
	table := "Configurations"
	if ss.src.DriverName() == model.DatabaseDriverPostgres {
		table = "configurations"
	}

	var value sql.NullString
	err := ss.replica.Get(&value, "SELECT value FROM "+table+" WHERE active = true")
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !value.Valid {
		return nil, nil
	}
	return []byte(value.String), nil
}
