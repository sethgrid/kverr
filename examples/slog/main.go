// Package main demonstrates kverr with the standard library's log/slog logger.
//
// It shows two complementary extraction patterns:
//
//  1. kverr.Args(err) — flat variadic form, works with slog.Error/Info/etc.
//  2. slog.LogValuer — pass the *Error directly; slog calls LogValue() automatically.
//
// It also verifies that errors.Is still works through a kverr boundary.
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/sethgrid/kverr"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	err := fetchUser("u-123", "req-abc")
	if err != nil {
		// Pattern 1: flat variadic — extracts KV as alternating key/val args.
		logger.Error("request failed (Args)", kverr.Args(err, "pattern", "Args")...)

		// Pattern 2: pass the error directly — slog calls LogValue() which emits
		// the KV pairs as a structured group named "err".
		logger.Error("request failed (LogValuer)", "err", err, "pattern", "LogValuer")

		// errors.Is still works through the kverr boundary.
		fmt.Println("errors.Is sql.ErrNoRows:", errors.Is(err, sql.ErrNoRows))
	}
}

// fetchUser simulates a multi-layer call stack that annotates the error with
// context at each level as it bubbles up.
func fetchUser(userID, requestID string) error {
	if err := queryDB(userID); err != nil {
		// Add request-level context at this layer.
		return kverr.New(err, slog.String("request_id", requestID), slog.String("handler", "fetchUser"))
	}
	return nil
}

func queryDB(userID string) error {
	// Simulate a not-found result. Wrap the stdlib sentinel with kverr so the
	// caller can both detect it with errors.Is and extract structured context.
	return kverr.New(sql.ErrNoRows, "user_id", userID, "table", "users")
}
