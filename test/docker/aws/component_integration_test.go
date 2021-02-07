// +build integration

package aws

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/beatlabs/patron/component/sqs"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Component_Run(t *testing.T) {
	const queueName = "test-component-run"
	const correlationID = "123"

	api, err := createSQSAPI(runtime.getSQSEndpoint())
	require.NoError(t, err)
	queue, err := createSQSQueue(api, queueName)
	require.NoError(t, err)

	_ = sendMessage(t, api, correlationID, queue, "1", "2", "3")

	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)

	proc := processor{t: t}
	cmp, err := sqs.New("test-component", queueName, api, proc.process)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	ctx, cnl := context.WithCancel(context.Background())

	go func() {
		require.NoError(t, cmp.Run(ctx))
		wg.Done()
	}()

	time.Sleep(2 * time.Second)
	cnl()
	wg.Wait()

	assert.True(t, len(mtr.FinishedSpans()) > 0)
}

type processor struct {
	t *testing.T
}

func (p processor) process(_ context.Context, batch sqs.Batch) {
	_, err := batch.ACK()
	require.NoError(p.t, err)
}
