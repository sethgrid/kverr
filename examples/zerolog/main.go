// Package main demonstrates kverr with github.com/rs/zerolog.
//
// zerolog uses a fluent builder API (event.Str, event.Interface, etc.).
// kverr.Map(err) feeds the builder loop shown below. The pattern keeps
// zerolog's zero-allocation characteristics intact for string and numeric
// values when you replace Interface with the appropriately typed method.
//
// It also verifies that errors.Is still works through a kverr boundary.
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/sethgrid/kverr"
)

func main() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	err := fetchUser("u-789", "req-def")
	if err != nil {
		// Build the zerolog event, attaching each KV pair from the error chain.
		ev := log.Error().Err(err)
		for k, v := range kverr.Map(err) {
			ev = ev.Interface(k, v)
		}
		ev.Msg("request failed")

		fmt.Println("errors.Is sql.ErrNoRows:", errors.Is(err, sql.ErrNoRows))
	}
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
