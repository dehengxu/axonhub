package sqlite

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"modernc.org/sqlite"
)

// https://github.com/ent/ent/discussions/1667
type sqliteDriver struct {
	*sqlite.Driver
}

func (d sqliteDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return conn, err
	}

	//nolint:forcetypeassert
	c := conn.(interface {
		Exec(stmt string, args []driver.Value) (driver.Result, error)
	})

	// Enable foreign keys first
	if _, err := c.Exec("PRAGMA foreign_keys = on;", nil); err != nil {
		if err := conn.Close(); err != nil {
			return nil, fmt.Errorf("failed to close connection: %w", err)
		}

		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Apply SQLite optimizations
	if err := setupSQLiteOptimizations(c); err != nil {
		if err := conn.Close(); err != nil {
			return nil, fmt.Errorf("failed to close connection: %w", err)
		}

		return nil, fmt.Errorf("failed to setup sqlite optimizations: %w", err)
	}

	return conn, nil
}

// setupSQLiteOptimizations applies performance and reliability optimizations
func setupSQLiteOptimizations(conn interface {
	Exec(stmt string, args []driver.Value) (driver.Result, error)
}) error {
	pragmas := []string{
		"PRAGMA synchronous = NORMAL",           // Balance performance and safety
		"PRAGMA cache_size = 10000",            // 10MB cache (pages * 1KB)
		"PRAGMA temp_store = memory",           // Store temp tables in memory
		"PRAGMA mmap_size = 268435456",         // 256MB memory-mapped I/O
		"PRAGMA journal_size_limit = 65536",      // 64MB journal size limit
		"PRAGMA wal_autocheckpoint = 1000",       // Auto checkpoint every 1000 pages
	}

	for _, pragma := range pragmas {
		if _, err := conn.Exec(pragma, nil); err != nil {
			return fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
		}
	}

	return nil
}

// SetupSQLiteConnectionPool configures the database connection pool for better concurrency
func SetupSQLiteConnectionPool(db *sql.DB) error {
	// Connection pool settings for SQLite
	db.SetMaxOpenConns(25)                    // Increase max connections for better concurrency
	db.SetMaxIdleConns(5)                     // Keep some idle connections ready
	db.SetConnMaxLifetime(5 * time.Minute)        // Don't keep connections alive too long
	db.SetConnMaxIdleTime(2 * time.Minute)        // Close idle connections after 2 minutes

	return nil
}

// HandleTransactionError handles common SQLite transaction errors
func HandleTransactionError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())

	// Transaction already committed - not really an error
	if strings.Contains(errMsg, "transaction has already been committed") {
		return nil
	}

	// Transaction already rolled back - not really an error
	if strings.Contains(errMsg, "transaction has already been rolled back") {
		return nil
	}

	// Database locked - can retry
	if strings.Contains(errMsg, "database is locked") {
		time.Sleep(100 * time.Millisecond)
		return fmt.Errorf("database locked, retry recommended: %w", err)
	}

	// Database busy - can retry
	if strings.Contains(errMsg, "database is busy") {
		time.Sleep(200 * time.Millisecond)
		return fmt.Errorf("database busy, retry recommended: %w", err)
	}

	return err
}

// CheckTransactionState checks if a transaction is still valid
func CheckTransactionState(tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	// Try a simple query to check transaction state
	rows, err := tx.Query("SELECT 1", nil)
	if err != nil {
		return err
	}

	defer rows.Close()
	return nil
}

func init() {
	sql.Register("sqlite3", sqliteDriver{Driver: &sqlite.Driver{}})
}
