package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// readActiveConfigurationDSN reads the active configuration row from the
// database named by MM_CONFIG. That database can be separate from the
// application database the plugin's SQLStore uses.
func readActiveConfigurationDSN(dsn string) ([]byte, error) {
	driver, dataSource, table, err := configDSN(dsn)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driver, dataSource)
	if err != nil {
		return nil, fmt.Errorf("open MM_CONFIG database: %w", err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query, err := activeConfigQuery(table)
	if err != nil {
		return nil, err
	}
	var value sql.NullString
	err = db.QueryRowContext(ctx, query).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read active configuration from MM_CONFIG database: %w", err)
	}
	if !value.Valid {
		return nil, nil
	}
	return []byte(value.String), nil
}

// activeConfigQuery uses a fixed statement per table. The table name is chosen
// by the driver, and it is not interpolated from the DSN.
func activeConfigQuery(table string) (string, error) {
	switch table {
	case "configurations":
		return "SELECT value FROM configurations WHERE active = true", nil
	case "Configurations":
		return "SELECT value FROM Configurations WHERE active = true", nil
	default:
		return "", fmt.Errorf("unsupported configuration table %q", table)
	}
}

func configDSN(dsn string) (driver, dataSource, table string, err error) {
	parts := strings.SplitN(dsn, "://", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", "", "", fmt.Errorf("MM_CONFIG is not a database DSN")
	}
	switch parts[0] {
	case "postgres", "postgresql":
		// lib/pq accepts the URL form, including the scheme.
		return "postgres", dsn, "configurations", nil
	case "mysql":
		// The MySQL driver does not accept a mysql:// scheme.
		return "mysql", parts[1], "Configurations", nil
	default:
		return "", "", "", fmt.Errorf("MM_CONFIG database scheme %q is not supported", parts[0])
	}
}
