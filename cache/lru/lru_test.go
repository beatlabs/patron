package lru

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		err     string
		size    int
		wantErr bool
	}{
		"negative size": {size: -1, wantErr: true, err: "must provide a positive size"},
		"zero size":     {size: 0, wantErr: true, err: "must provide a positive size"},
		"positive size": {size: 1024, wantErr: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c, err := New[string, string](tt.size, "test")
			if tt.wantErr {
				assert.Nil(t, c)
				assert.EqualError(t, err, tt.err)
			} else {
				assert.NotNil(t, c)
				require.NoError(t, err)
			}
		})
	}
}

func TestNewWithEvict(t *testing.T) {
	tests := map[string]struct {
		err     string
		size    int
		wantErr bool
	}{
		"negative size": {size: -1, wantErr: true, err: "must provide a positive size"},
		"zero size":     {size: 0, wantErr: true, err: "must provide a positive size"},
		"positive size": {size: 1024, wantErr: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c, err := NewWithEvict[string, string](tt.size, "test", func(_, _ string) {})
			if tt.wantErr {
				assert.Nil(t, c)
				assert.EqualError(t, err, tt.err)
			} else {
				assert.NotNil(t, c)
				require.NoError(t, err)
			}
		})
	}
}

func TestCacheOperations(t *testing.T) {
	c, err := NewWithEvict[string, string](10, "test", func(_, _ string) {})
	assert.NotNil(t, c)
	require.NoError(t, err)

	k, v := "foo", "bar"
	ctx := context.Background()

	t.Run("testGetEmpty", func(t *testing.T) {
		res, ok, err := c.Get(ctx, k)
		assert.Nil(t, res)
		assert.False(t, ok)
		require.NoError(t, err)
	})

	t.Run("testSetGet", func(t *testing.T) {
		err = c.Set(ctx, k, v)
		require.NoError(t, err)
		res, ok, err := c.Get(ctx, k)
		assert.Equal(t, v, res)
		assert.True(t, ok)
		require.NoError(t, err)
	})

	t.Run("testRemove", func(t *testing.T) {
		err = c.Remove(ctx, k)
		require.NoError(t, err)
		res, ok, err := c.Get(ctx, k)
		assert.Nil(t, res)
		assert.False(t, ok)
		require.NoError(t, err)
	})

	t.Run("testPurge", func(t *testing.T) {
		err = c.Set(ctx, "key1", "val1")
		require.NoError(t, err)
		err = c.Set(ctx, "key2", "val2")
		require.NoError(t, err)
		err = c.Set(ctx, "key3", "val3")
		require.NoError(t, err)

		assert.Equal(t, 3, c.cache.Len())
		err = c.Purge(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, c.cache.Len())
	})
}

func BenchmarkCache_Set(b *testing.B) {
	c, _ := NewWithEvict(b.N, "test", func(_, _ int) {})
	ctx := context.Background()

	for b.Loop() {
		c.Set(ctx, 1, 1)
	}
}

func BenchmarkCache_Get(b *testing.B) {
	c, _ := NewWithEvict(b.N, "test", func(_, _ int) {})
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		c.Set(ctx, i, i)
	}

	for b.Loop() {
		c.Get(ctx, 1)
	}
}
