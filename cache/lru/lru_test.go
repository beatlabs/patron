package lru

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
		err     string
	}{
		{name: "negative size", size: -1, wantErr: true, err: "Must provide a positive size"},
		{name: "zero size", size: 0, wantErr: true, err: "Must provide a positive size"},
		{name: "positive size", size: 1024, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.size)
			if tt.wantErr {
				assert.Nil(t, c)
				assert.EqualError(t, err, tt.err)
			} else {
				assert.NotNil(t, c)
				assert.NoError(t, err)
			}
		})
	}
}

func TestCacheOperations(t *testing.T) {
	c, err := New(10)
	assert.NotNil(t, c)
	assert.NoError(t, err)

	k, v := "foo", "bar"

	t.Run("testGetEmpty", func(t *testing.T) {
		res, ok, err := c.Get(k)
		assert.Nil(t, res)
		assert.False(t, ok)
		assert.NoError(t, err)
	})

	t.Run("testSetGet", func(t *testing.T) {
		err = c.Set(k, v)
		assert.NoError(t, err)
		res, ok, err := c.Get(k)
		assert.Equal(t, v, res)
		assert.True(t, ok)
		assert.NoError(t, err)
	})

	t.Run("testRemove", func(t *testing.T) {
		err = c.Remove(k)
		assert.NoError(t, err)
		res, ok, err := c.Get(k)
		assert.Nil(t, res)
		assert.False(t, ok)
		assert.NoError(t, err)
	})

	t.Run("testPurge", func(t *testing.T) {
		err = c.Set("key1", "val1")
		assert.NoError(t, err)
		err = c.Set("key2", "val2")
		assert.NoError(t, err)
		err = c.Set("key3", "val3")
		assert.NoError(t, err)

		assert.Equal(t, c.cache.Len(), 3)
		err = c.Purge()
		assert.NoError(t, err)
		assert.Equal(t, c.cache.Len(), 0)
	})
}
