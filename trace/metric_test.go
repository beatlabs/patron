package trace

import (
	"context"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounter_Add(t *testing.T) {
	type fields struct {
		counter prometheus.Counter
	}
	type args struct {
		count float64
	}
	test := struct {
		name   string
		fields fields
		args   args
	}{
		name: "test-add-counter",
		fields: fields{
			counter: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "test_counter",
				},
				[]string{"name"},
			).WithLabelValues("test"),
		},
		args: args{
			count: 2,
		},
	}
	t.Run(test.name, func(t *testing.T) {
		c := Counter{
			Counter: test.fields.counter,
		}
		c.Add(context.Background(), test.args.count)
		assert.Equal(t, test.args.count, testutil.ToFloat64(c))
		c.Add(context.Background(), test.args.count)
		assert.Equal(t, 2*test.args.count, testutil.ToFloat64(c))
	})
}

func TestCounter_Inc(t *testing.T) {
	type fields struct {
		counter prometheus.Counter
	}
	type args struct {
		count int
	}
	test := struct {
		name   string
		fields fields
		args   args
	}{
		name: "test-inc-counter",
		fields: fields{
			counter: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "test_counter",
				},
				[]string{"name"},
			).WithLabelValues("test"),
		},
		args: args{
			count: 2,
		},
	}
	t.Run(test.name, func(t *testing.T) {
		c := Counter{
			Counter: test.fields.counter,
		}
		c.Inc(context.Background())
		assert.Equal(t, float64(1), testutil.ToFloat64(c))
		c.Inc(context.Background())
		assert.Equal(t, float64(2), testutil.ToFloat64(c))
	})
}

func TestHistogram_Observe(t *testing.T) {
	type fields struct {
		histogram *prometheus.HistogramVec
	}
	type args struct {
		val float64
	}
	test := struct {
		name   string
		fields fields
		args   args
	}{
		name: "test-observe-histogram",
		fields: fields{
			histogram: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name: "test_histogram",
				},
				[]string{"name"},
			),
		},
		args: args{
			val: 2,
		},
	}
	t.Run(test.name, func(t *testing.T) {
		h := Histogram{
			Observer: test.fields.histogram.WithLabelValues("test"),
		}
		h.Observe(context.Background(), test.args.val)
		actualVal, err := sampleSum(test.fields.histogram)
		require.Nil(t, err)
		assert.Equal(t, test.args.val, actualVal)
		h.Observe(context.Background(), test.args.val)
		actualVal, err = sampleSum(test.fields.histogram)
		require.Nil(t, err)
		assert.Equal(t, 2*test.args.val, actualVal)
	})
}

func sampleSum(c prometheus.Collector) (float64, error) {
	var (
		m      prometheus.Metric
		mCount int
		mChan  = make(chan prometheus.Metric)
		done   = make(chan struct{})
	)

	go func() {
		for m = range mChan {
			mCount++
		}
		close(done)
	}()

	c.Collect(mChan)
	close(mChan)
	<-done

	if mCount != 1 {
		return -1, fmt.Errorf("collected %d metrics instead of exactly 1", mCount)
	}

	pb := &dto.Metric{}
	m.Write(pb)

	if pb.Histogram != nil {
		return *pb.Histogram.SampleSum, nil
	}
	return -1, fmt.Errorf("collected a non-histogram metric: %s", pb)
}
