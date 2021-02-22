// +build integration

package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/beatlabs/patron/client/redis"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var runtime *redisRuntime

func TestMain(m *testing.M) {
	var err error
	runtime, err = create(60 * time.Second)
	if err != nil {
		fmt.Printf("could not create mysql runtime: %v\n", err)
		os.Exit(1)
	}
	defer func() {
	}()
	exitCode := m.Run()

	ee := runtime.Teardown()
	if len(ee) > 0 {
		for _, err = range ee {
			fmt.Printf("could not tear down containers: %v\n", err)
		}
	}
	os.Exit(exitCode)
}

func TestName(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	defer mtr.Reset()
	dsn, err := runtime.DSN()
	require.NoError(t, err)

	cl := redis.New(redis.Options{
		Addr:     dsn,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	cmd := cl.Set(context.Background(), "key", "value", 0)
	res, err := cmd.Result()
	assert.NoError(t, err)
	assert.Equal(t, res, "OK")
	assert.Len(t, mtr.FinishedSpans(), 1)
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["component"], "redis")
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["error"], false)
	assert.Regexp(t, `:\d+`, mtr.FinishedSpans()[0].Tags()["db.instance"])
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["db.statement"], "set key value")
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["db.type"], "kv")
}
