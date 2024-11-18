package kverr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKVErr(t *testing.T) {
	err := skipLevelKVErr()

	args := YoinkArgs(err)
	require.Len(t, args, 2)
	assert.Equal(t, args[0], "kv_present")
	assert.Equal(t, args[1], true)
}

func TestKVErrAppend(t *testing.T) {
	err := skipLevelKVErr()
	err = New(fmt.Errorf("wrap some more: %w", err), "another_key", "another_value")
	err = fmt.Errorf("another wrap for good measure: %w", err)

	args := YoinkArgs(err)
	require.Len(t, args, 4)
	// arg order is not guaranteed if multiple keys exist in wrapped kv error due to map random access
	assert.Contains(t, args, "kv_present")
	assert.Contains(t, args, true)
	assert.Contains(t, args, "another_key")
	assert.Contains(t, args, "another_value")
}

func skipLevelKVErr() error {
	if err := returnKVErr(); err != nil {
		return fmt.Errorf("oh noes, err: %w", err)
	}
	return nil
}

func returnKVErr() error {
	return New(fmt.Errorf("root error"), "kv_present", true)
}
