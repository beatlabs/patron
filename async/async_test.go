package async

import (
	"testing"

	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/stretchr/testify/assert"
)

func TestDetermineDecoder(t *testing.T) {
	type args struct {
		contentType string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success json", args{contentType: json.Type}, false},
		{"success protobuf", args{contentType: protobuf.Type}, false},
		{"failure", args{contentType: "XXX"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetermineDecoder(tt.args.contentType)
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
