package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mattermost/mattermost-server/v6/model"
)

var (
	dbUser                     = "mmuser"
	dbPassword                 = "mostest"
	ErrUnsupportedDatabaseType = errors.New("unsupported database driver name")
)

type TearDownFunc func(ctx context.Context) error

// CreateTestDB instantiates a database container (postres or mysql) and returns the connection
// string needed to connect to that database. A teardown function is also returned to close the
// container and free resources.
func CreateTestDB(ctx context.Context, driverName string, databaseName string) (string, TearDownFunc, error) {
	var connStr string
	var tearDown TearDownFunc
	var err error

	switch driverName {
	case model.DatabaseDriverPostgres:
		connStr, tearDown, err = createTestDBPostgres(ctx, databaseName)
	case model.DatabaseDriverMysql:
		connStr, tearDown, err = createTestDBMySQL(ctx, databaseName)
	default:
		return "", nil, ErrUnsupportedDatabaseType
	}

	if err == nil {
		os.Setenv("TEST_DATABASE_POSTGRESQL_DSN", connStr)
	}

	return connStr, tearDown, err
}

func createTestDBMySQL(ctx context.Context, databaseName string) (string, TearDownFunc, error) {
	mysqlContainer, err := mysql.RunContainer(ctx,
		testcontainers.WithImage("mysql:8.0.32"),
		mysql.WithDatabase(databaseName),
		mysql.WithUsername(dbUser),
		mysql.WithPassword(dbPassword),
	)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to start mysql container")
	}

	tearDown := func(ctx context.Context) error {
		return mysqlContainer.Terminate(ctx)
	}

	connStr, err := mysqlContainer.ConnectionString(ctx, "readTimeout=30s", "writeTimeout=30s", "charset=utf8mb4,utf8")
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot generate connection string for mysql")
	}

	return connStr, tearDown, nil
}

func createTestDBPostgres(ctx context.Context, databaseName string) (string, TearDownFunc, error) {
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("docker.io/library/postgres:15.2-alpine"),
		postgres.WithDatabase(databaseName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second)),
	)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to start postgres container")
	}

	tearDown := func(ctx context.Context) error {
		return postgresContainer.Terminate(ctx)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable", "connect_timeout=30")
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot generate connection string for postgres")
	}

	port, err := postgresContainer.MappedPort(context.TODO(), "5432/tcp")
	if err != nil {
		return "", nil, errors.Wrap(err, "mapped port")
	}

	fmt.Println("mapped port: ", port)

	return connStr, tearDown, nil
}

// CreateMinio instantiates a minio container and returns the connection string needed to connect to it.
// A teardown function is also returned to close the container and free resources.
func CreateMinio(ctx context.Context) (string, TearDownFunc, error) {
	minioContainer, err := minio.RunContainer(
		ctx,
		testcontainers.WithImage("minio/minio:RELEASE.2024-01-16T16-07-38Z"),
		// Create default bucket
		testcontainers.WithStartupCommand(testcontainers.NewRawCommand([]string{"mkdir", "/data/" + model.MinioBucket})),
		testcontainers.WithEnv(map[string]string{
			"MINIO_ROOT_USER":     model.MinioAccessKey,
			"MINIO_ROOT_PASSWORD": model.MinioSecretKey,
		}),
	)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to start minio container")
	}

	tearDown := func(ctx context.Context) error {
		return minioContainer.Terminate(ctx)
	}

	connStr, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		return "", nil, errors.Wrap(err, "cannot generate connection string for minio")
	}

	return connStr, tearDown, nil
}
