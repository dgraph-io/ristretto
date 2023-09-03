package ristretto

import (
	"context"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLockedCallerDo(t *testing.T) {
	caller := newLockedCaller()

	v, err := caller.do(context.Background(), "key", 0, func(ctx context.Context, key interface{}) (interface{}, error) {
		return "foo", nil
	})

	require.NoError(t, err)
	require.Equal(t, "foo", v)
}

func TestLockedCallerDoError(t *testing.T) {
	caller := newLockedCaller()

	errTest := errors.New("test")
	v, err := caller.do(context.Background(), "key", 0, func(ctx context.Context, key interface{}) (interface{}, error) {
		return nil, errTest
	})

	require.Equal(t, errTest, err)
	require.Nil(t, v)
}

func TestLockedCallerDoDeDuplicated(t *testing.T) {
	caller := newLockedCaller()

	ch := make(chan interface{})
	callCount := int32(0)
	fn := func(ctx context.Context, key interface{}) (interface{}, error) {
		atomic.AddInt32(&callCount, 1)
		return <-ch, nil
	}

	const n = 10
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			v, err := caller.do(context.Background(), "key", 0, fn)

			require.NoError(t, err)
			require.Equal(t, "foo", v)
		}()
	}

	time.Sleep(100 * time.Millisecond) // let goroutines blocked

	ch <- "foo"

	wg.Wait()

	require.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestShardedCallerDo(t *testing.T) {
	caller := newShardedCaller()

	v, err := caller.Do(context.Background(), "key", 0, func(ctx context.Context, key interface{}) (interface{}, error) {
		return "foo", nil
	})

	require.NoError(t, err)
	require.Equal(t, "foo", v)
}