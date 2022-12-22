package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTest    = errors.New("test error")
	testResult = "test result"
)

func TestNew(t *testing.T) {
	type args struct {
		attempts int
		delay    time.Duration
	}
	tests := map[string]struct {
		name    string
		args    args
		wantErr bool
	}{
		"success":          {args: args{attempts: 3, delay: 3 * time.Second}, wantErr: false},
		"invalid attempts": {args: args{attempts: -1, delay: 3 * time.Second}, wantErr: true},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			got, err := New(tt.args.attempts, tt.args.delay)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_Retry_Execute(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		attempts           int
		delay              time.Duration
		action             mockAction
		expectedExecutions int
		expectErr          bool
	}{
		"instant success": {
			attempts:           3,
			action:             mockAction{errors: 0},
			expectedExecutions: 1,
		},
		"instant success with delay": {
			attempts:           3,
			delay:              500 * time.Millisecond,
			action:             mockAction{errors: 0},
			expectedExecutions: 1,
		},
		"success without delay after an error": {
			attempts:           3,
			action:             mockAction{errors: 1},
			expectedExecutions: 2,
		},
		"success with delay after an error": {
			attempts:           3,
			delay:              500 * time.Millisecond,
			action:             mockAction{errors: 1},
			expectedExecutions: 2,
		},
		"error after exceeding one failed attempt": {
			attempts:           2,
			action:             mockAction{errors: 2},
			expectedExecutions: 2,
			expectErr:          true,
		},
		"error after exceeding number of failed attempts": {
			attempts:           3,
			action:             mockAction{errors: 3},
			expectedExecutions: 3,
			expectErr:          true,
		},
	}
	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r, err := New(tt.attempts, tt.delay)
			require.NoError(t, err)

			start := time.Now()
			res, err := r.Execute(func() (interface{}, error) {
				return tt.action.Execute()
			})
			elapsed := time.Since(start)

			if tt.expectErr {
				assert.Equal(t, err, errTest)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testResult, res)
			}

			assert.Equal(t, tt.expectedExecutions, tt.action.executions)

			// Assert that the total time takes into account the delay between attempts
			assert.True(t, elapsed > tt.delay*time.Duration(tt.expectedExecutions-1))
		})
	}
}

type mockAction struct {
	errors     int
	executions int
}

func (ma *mockAction) Execute() (string, error) {
	defer func() {
		ma.errors--
		ma.executions++
	}()
	if ma.errors > 0 {
		return "", errTest
	}
	return testResult, nil
}

var err error

func BenchmarkRetry_Execute(b *testing.B) {
	var r *Retry
	r, err = New(3, 0)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = r.Execute(testSuccessAction)
	}
}

func testSuccessAction() (interface{}, error) {
	return "test", nil
}
