package transaction

import (
	"context"
	"database/sql"
	"strings"

	"entgo.io/ent"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/pkg/sqlite"
	"go.uber.org/zap"
)

// TransactionManager provides safe transaction handling with proper state management
type TransactionManager struct {
	client *ent.Client
	logger *zap.Logger
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(client *ent.Client, logger *zap.Logger) *TransactionManager {
	return &TransactionManager{
		client: client,
		logger: logger,
	}
}

// WithTransaction executes a function within a transaction with proper error handling
// and state management to prevent duplicate commits/rollbacks
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(*ent.Tx) error) error {
	// Use serializable isolation for better consistency in high concurrency
	tx, err := tm.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if err != nil {
		return tm.handleError(ctx, "failed to begin transaction", err)
	}

	// Ensure transaction is handled exactly once
	var committed bool
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				tm.logger.Error(ctx, "Failed to rollback transaction",
					zap.String("error", rollbackErr.Error()))
			}
		}
	}()

	// Execute business logic within transaction
	if err := fn(tx); err != nil {
		// Handle transaction errors gracefully
		if handledErr := tm.handleError(ctx, "transaction execution failed", err); handledErr == nil {
			return nil
		}
		return err
	}

	// Mark as committed before actual commit
	committed = true
	if err := tx.Commit(); err != nil {
		return tm.handleError(ctx, "failed to commit transaction", err)
	}

	return nil
}

// WithReadOnlyTransaction executes a function within a read-only transaction
func (tm *TransactionManager) WithReadOnlyTransaction(ctx context.Context, fn func(*ent.Client) error) error {
	tx, err := tm.client.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return tm.handleError(ctx, "failed to begin read-only transaction", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			tm.logger.Error(ctx, "Failed to rollback read-only transaction",
				zap.String("error", err.Error()))
		}
	}()

	// Execute business logic with transaction client
	return fn(tx.Client())
}

// handleError processes transaction errors with context and applies SQLite error handling
func (tm *TransactionManager) handleError(ctx context.Context, message string, err error) error {
	if err == nil {
		tm.logger.Warn(ctx, message+": no error provided")
		return nil
	}

	// Apply SQLite-specific error handling
	if handledErr := sqlite.HandleTransactionError(err); handledErr == nil {
		// Error was handled (not really an error), return nil
		tm.logger.Debug(ctx, message+": error handled gracefully", zap.String("error", err.Error()))
		return nil
	}

	tm.logger.Error(ctx, message, zap.String("error", err.Error()))
	return err
}