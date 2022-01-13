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
	t.Parallel()
	type fields struct {
		counter prometheus.Counter
	}
	type args struct {
		count float64
	}
	tests := map[string]struct {
		fields        fields
		args          args
		expectedVal   float64
		expectedPanic bool
	}{
		"test-add-counter": {
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
			expectedVal:   2,
			expectedPanic: false,
		},
		"test-try-to-decrease-counter": {
			fields: fields{
				counter: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "test_counter",
					},
					[]string{"name"},
				).WithLabelValues("test"),
			},
			args: args{
				count: -2,
			},
			expectedPanic: true,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if tt.expectedPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Add method did not panic.")
					}
				}()
			}
			c := Counter{
				Counter: tt.fields.counter,
			}
			c.Add(context.Background(), tt.args.count)
			if tt.expectedPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Add method did not panic.")
					}
				}()
			} else {
				assert.Equal(t, tt.expectedVal, testutil.ToFloat64(c))
				c.Add(context.Background(), tt.args.count)
				assert.Equal(t, 2*tt.expectedVal, testutil.ToFloat64(c))
			}
		})
	}
}

func TestCounter_Inc(t *testing.T) {
	t.Parallel()
	type fields struct {
		counter prometheus.Counter
	}
	type args struct {
		count int
	}
	tests := map[string]struct {
		fields      fields
		args        args
		expectedVal float64
	}{
		"test-inc-counter": {
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
			expectedVal: 1,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := Counter{
				Counter: tt.fields.counter,
			}
			c.Inc(context.Background())
			assert.Equal(t, tt.expectedVal, testutil.ToFloat64(c))
			c.Inc(context.Background())
			assert.Equal(t, 2*tt.expectedVal, testutil.ToFloat64(c))
		})
	}
}

func TestHistogram_Observe(t *testing.T) {
	t.Parallel()
	type fields struct {
		histogram *prometheus.HistogramVec
	}
	type args struct {
		val float64
	}
	tests := map[string]struct {
		fields      fields
		args        args
		expectedVal float64
	}{
		"test-observe-histogram": {
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
			expectedVal: 2,
		},
		"test-observe-histogram-negative-value": {
			fields: fields{
				histogram: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name: "test_histogram",
					},
					[]string{"name"},
				),
			},
			args: args{
				val: -2,
			},
			expectedVal: -2,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := Histogram{
				Observer: tt.fields.histogram.WithLabelValues("test"),
			}
			h.Observe(context.Background(), tt.args.val)
			actualVal, err := sampleSum(tt.fields.histogram)
			require.Nil(t, err)
			assert.Equal(t, tt.args.val, actualVal)
			h.Observe(context.Background(), tt.args.val)
			actualVal, err = sampleSum(tt.fields.histogram)
			require.Nil(t, err)
			assert.Equal(t, 2*tt.args.val, actualVal)
		})
	}
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
	_ = m.Write(pb)

	if pb.Histogram != nil {
		return *pb.Histogram.SampleSum, nil
	}
	return -1, fmt.Errorf("collected a non-histogram metric: %s", pb)
}
