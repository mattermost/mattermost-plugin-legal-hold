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
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var value sql.NullString
	err = db.QueryRowContext(ctx, "SELECT value FROM "+table+" WHERE active = true").Scan(&value)
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
