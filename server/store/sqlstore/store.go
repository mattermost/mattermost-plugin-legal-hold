package sqlstore

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/mattermost/mattermost/server/public/model"
)

type Source interface {
	GetMasterDB() (*sql.DB, error)
	GetReplicaDB() (*sql.DB, error)
	DriverName() string
}

type Logger interface {
	Error(message string, keyValuePairs ...interface{})
	Warn(message string, keyValuePairs ...interface{})
	Info(message string, keyValuePairs ...interface{})
	Debug(message string, keyValuePairs ...interface{})
}

type SQLStore struct {
	src            Source
	master         *sqlx.DB
	replica        *sqlx.DB
	masterBuilder  sq.StatementBuilderType
	replicaBuilder sq.StatementBuilderType
	logger         Logger
}

// New constructs a new instance of SQLStore.
func New(src Source, logger Logger) (*SQLStore, error) {
	var master, replica *sqlx.DB

	masterDB, err := src.GetMasterDB()
	if err != nil {
		return nil, err
	}
	master = sqlx.NewDb(masterDB, src.DriverName())

	replicaDB, err := src.GetReplicaDB()
	if err != nil {
		return nil, err
	}
	replica = sqlx.NewDb(replicaDB, src.DriverName())

	masterBuilder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	if src.DriverName() == model.DatabaseDriverPostgres {
		masterBuilder = masterBuilder.PlaceholderFormat(sq.Dollar)
	}

	if src.DriverName() == model.DatabaseDriverMysql {
		master.MapperFunc(func(s string) string { return s })
	}

	masterBuilder = masterBuilder.RunWith(master)

	replicaBuilder := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	if src.DriverName() == model.DatabaseDriverPostgres {
		replicaBuilder = replicaBuilder.PlaceholderFormat(sq.Dollar)
	}

	if src.DriverName() == model.DatabaseDriverMysql {
		replica.MapperFunc(func(s string) string { return s })
	}

	replicaBuilder = replicaBuilder.RunWith(replica)

	return &SQLStore{
		src,
		master,
		replica,
		masterBuilder,
		replicaBuilder,
		logger,
	}, nil
}
