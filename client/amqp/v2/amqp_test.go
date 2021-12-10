package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	type args struct {
		url string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"fail, missing url": {args: args{}, expectedErr: "url is required"},
	}
	for name, tt := range tests {
		tst := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := New(tst.args.url)
			if tst.expectedErr != "" {
				assert.EqualError(t, err, tst.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}
