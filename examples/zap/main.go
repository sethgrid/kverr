// Package main demonstrates kverr with go.uber.org/zap.
//
// zap uses typed Field values (zap.String, zap.Int, etc.) rather than a
// variadic any API, so the integration point is kverr.Map(err). A small local
// helper converts the map to []zap.Field. This pattern composes cleanly with
// zap's zero-allocation design: use zap.Any for values whose type you don't
// need to optimise, or unwrap to the concrete type for hot paths.
//
// It also verifies that errors.Is still works through a kverr boundary.
package main

import (
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/sethgrid/kverr"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	err := fetchUser("u-456", "req-xyz")
	if err != nil {
		// Convert kverr context to zap fields and append zap.Error for the
		// error message itself.
		fields := append(zapFields(err), zap.Error(err))
		logger.Error("request failed", fields...)

		fmt.Println("errors.Is sql.ErrNoRows:", errors.Is(err, sql.ErrNoRows))
	}
}

// zapFields converts all KV pairs carried by err into []zap.Field.
// zap.Any handles any value type via reflection; replace with a typed
// constructor (zap.String, zap.Int64, …) in hot paths.
func zapFields(err error) []zap.Field {
	m := kverr.Map(err)
	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		fields = append(fields, zap.Any(k, v))
	}
	return fields
}

func fetchUser(userID, requestID string) error {
	if err := queryDB(userID); err != nil {
		return kverr.New(err, "request_id", requestID, "handler", "fetchUser")
	}
	return nil
}

func queryDB(userID string) error {
	return kverr.New(sql.ErrNoRows, "user_id", userID, "table", "users")
}
