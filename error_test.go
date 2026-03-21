package kverr

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sentinel errors used across tests to verify errors.Is / errors.As.
var errNotFound = errors.New("not found")

type customErr struct{ code int }

func (c *customErr) Error() string { return fmt.Sprintf("custom error code=%d", c.code) }

// --- New ---

func TestNew_StringAnyPairs(t *testing.T) {
	err := New(errNotFound, "user_id", 42, "req_id", "abc")
	require.NotNil(t, err)
	m := Map(err)
	// slog.Any normalises int → int64 when building the slog.Value.
	assert.Equal(t, int64(42), m["user_id"])
	assert.Equal(t, "abc", m["req_id"])
}

func TestNew_SlogAttrs(t *testing.T) {
	err := New(errNotFound, slog.Int("user_id", 42), slog.String("req_id", "abc"))
	m := Map(err)
	assert.Equal(t, int64(42), m["user_id"]) // slog.Int stores as int64
	assert.Equal(t, "abc", m["req_id"])
}

func TestNew_MixedArgs(t *testing.T) {
	err := New(errNotFound, slog.String("user_id", "u1"), "req_id", "r1")
	m := Map(err)
	assert.Equal(t, "u1", m["user_id"])
	assert.Equal(t, "r1", m["req_id"])
}

func TestNew_NonStringKeySkipped(t *testing.T) {
	// A non-string, non-slog.Attr at any position is silently skipped; the
	// integer 42 is dropped without becoming a key or consuming its neighbour.
	err := New(errNotFound, "key", "val", 42)
	m := Map(err)
	assert.Equal(t, "val", m["key"])
	assert.Len(t, m, 1)
}

func TestNew_OddLengthArgsTrailingKeySkipped(t *testing.T) {
	err := New(errNotFound, "key", "val", "orphan")
	m := Map(err)
	assert.Equal(t, "val", m["key"])
	assert.NotContains(t, m, "orphan")
}

func TestNew_NilErr(t *testing.T) {
	err := New(nil, "key", "val")
	require.NotNil(t, err)
	assert.Nil(t, err.Err)
	assert.Equal(t, "val", Map(err)["key"])
}

func TestNew_NoArgs(t *testing.T) {
	err := New(errNotFound)
	require.NotNil(t, err)
	assert.Equal(t, errNotFound, err.Err)
	assert.Empty(t, Map(err))
}

// --- KV accumulation ---

func TestNew_MergesAncestorAttrs_DirectWrap(t *testing.T) {
	inner := New(errNotFound, "inner_key", "inner_val")
	outer := New(inner, "outer_key", "outer_val")

	m := Map(outer)
	assert.Equal(t, "inner_val", m["inner_key"])
	assert.Equal(t, "outer_val", m["outer_key"])
}

func TestNew_MergesAncestorAttrs_ThroughFmtErrorf(t *testing.T) {
	inner := New(errNotFound, "inner_key", "inner_val")
	wrapped := fmt.Errorf("mid layer: %w", inner)
	outer := New(wrapped, "outer_key", "outer_val")

	m := Map(outer)
	assert.Equal(t, "inner_val", m["inner_key"])
	assert.Equal(t, "outer_val", m["outer_key"])
}

func TestNew_MergesAcrossMultipleLayers(t *testing.T) {
	base := New(errNotFound, "a", "1")
	l2 := fmt.Errorf("l2: %w", base)
	l3 := New(l2, "b", "2")
	l4 := fmt.Errorf("l4: %w", l3)
	top := New(l4, "c", "3")

	m := Map(top)
	assert.Equal(t, "1", m["a"])
	assert.Equal(t, "2", m["b"])
	assert.Equal(t, "3", m["c"])
}

func TestNew_LaterKeyOverwritesEarlier(t *testing.T) {
	inner := New(errNotFound, "key", "original")
	outer := New(inner, "key", "overwritten")

	m := Map(outer)
	assert.Equal(t, "overwritten", m["key"])
}

// --- errors.Is / errors.As ---

func TestUnwrap_ErrorsIs_Sentinel(t *testing.T) {
	err := New(errNotFound, "user_id", 1)
	assert.True(t, errors.Is(err, errNotFound))
}

func TestUnwrap_ErrorsIs_ThroughFmtErrorf(t *testing.T) {
	inner := New(errNotFound, "user_id", 1)
	wrapped := fmt.Errorf("outer: %w", inner)
	assert.True(t, errors.Is(wrapped, errNotFound))
}

func TestUnwrap_ErrorsIs_StdlibSentinel(t *testing.T) {
	err := New(sql.ErrNoRows, "query", "select * from users")
	assert.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestUnwrap_ErrorsAs_CustomType(t *testing.T) {
	cause := &customErr{code: 404}
	err := New(cause, "user_id", 99)
	var target *customErr
	require.True(t, errors.As(err, &target))
	assert.Equal(t, 404, target.code)
}

func TestUnwrap_ErrorsAs_ThroughFmtErrorf(t *testing.T) {
	cause := &customErr{code: 500}
	inner := New(cause, "req_id", "r1")
	wrapped := fmt.Errorf("mid: %w", inner)
	outer := New(wrapped, "user_id", 1)
	var target *customErr
	require.True(t, errors.As(outer, &target))
	assert.Equal(t, 500, target.code)
}

// --- Error() ---

func TestError_ReturnsWrappedMessage(t *testing.T) {
	err := New(errNotFound, "key", "val")
	assert.Equal(t, "not found", err.Error())
}

func TestError_NilErrFallsBackToAttrs(t *testing.T) {
	err := New(nil, "key", "val")
	assert.Contains(t, err.Error(), "key")
}

// --- Args ---

func TestArgs_ExtractsPairs(t *testing.T) {
	err := New(errNotFound, "user_id", 42, "status", "active")
	args := Args(err)
	require.Len(t, args, 4)
	assert.Contains(t, args, "user_id")
	assert.Contains(t, args, "status")
	assert.Contains(t, args, "active")
}

func TestArgs_WithExtraStringAny(t *testing.T) {
	err := New(errNotFound, "user_id", 1)
	args := Args(err, "extra_key", "extra_val")
	assert.Contains(t, args, "extra_key")
	assert.Contains(t, args, "extra_val")
}

func TestArgs_WithExtraSlogAttr(t *testing.T) {
	err := New(errNotFound, "user_id", 1)
	args := Args(err, slog.String("extra_key", "extra_val"))
	assert.Contains(t, args, "extra_key")
	assert.Contains(t, args, "extra_val")
}

func TestArgs_NonKverrReturnsEmpty(t *testing.T) {
	args := Args(errNotFound)
	assert.NotNil(t, args)
	assert.Empty(t, args)
}

func TestArgs_NilReturnsEmpty(t *testing.T) {
	args := Args(nil)
	assert.NotNil(t, args)
	assert.Empty(t, args)
}

func TestArgs_ExtraOnlyNoKverr(t *testing.T) {
	args := Args(errNotFound, "k", "v")
	assert.Equal(t, []any{"k", "v"}, args)
}

// --- Map ---

func TestMap_ExtractsAllPairs(t *testing.T) {
	err := New(errNotFound, "a", 1, "b", "two")
	m := Map(err)
	assert.Equal(t, int64(1), m["a"]) // slog.Any normalises int → int64
	assert.Equal(t, "two", m["b"])
}

func TestMap_NonKverrReturnsEmptyMap(t *testing.T) {
	m := Map(errNotFound)
	assert.NotNil(t, m)
	assert.Empty(t, m)
}

func TestMap_NilReturnsEmptyMap(t *testing.T) {
	m := Map(nil)
	assert.NotNil(t, m)
	assert.Empty(t, m)
}

func TestMap_ThroughFmtErrorf(t *testing.T) {
	inner := New(errNotFound, "key", "val")
	wrapped := fmt.Errorf("wrap: %w", inner)
	m := Map(wrapped)
	assert.Equal(t, "val", m["key"])
}

// --- LogValue ---

func TestLogValue_ReturnsSlogGroup(t *testing.T) {
	err := New(errNotFound, slog.String("user_id", "u1"), slog.Int("count", 3))
	lv := err.LogValue()
	assert.Equal(t, slog.KindGroup, lv.Kind())
}

func TestLogValue_EmptyAttrs(t *testing.T) {
	err := New(errNotFound)
	lv := err.LogValue()
	assert.Equal(t, slog.KindGroup, lv.Kind())
	assert.Empty(t, lv.Group())
}

// --- Concurrency ---

func TestConcurrentNewAndArgs(t *testing.T) {
	base := New(errNotFound, "base", "val")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_ = New(base, "n", n)
		}(i)
		go func() {
			defer wg.Done()
			_ = Args(base)
		}()
	}
	wg.Wait()
}

func TestConcurrentMap(t *testing.T) {
	err := New(errNotFound, "key", "val")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := Map(err)
			assert.Equal(t, "val", m["key"])
		}()
	}
	wg.Wait()
}
