package mqtt

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	u, err := url.Parse("tcp://localhost:1388")
	require.NoError(t, err)
	type args struct {
		brokerURLs []*url.URL
		clientID   string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"missing broker urls": {args: args{brokerURLs: nil, clientID: "clientID"}, expectedErr: "no broker URLs provided"},
		"missing client id":   {args: args{brokerURLs: []*url.URL{u}, clientID: ""}, expectedErr: "no client id provided"},
		"success":             {args: args{brokerURLs: []*url.URL{u}, clientID: "clientID"}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := DefaultConfig(tt.args.brokerURLs, tt.args.clientID)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, got.ClientID, tt.args.clientID)
				assert.Equal(t, u, got.BrokerUrls[0])
				assert.Equal(t, uint16(30), got.KeepAlive)
				assert.Equal(t, 5*time.Second, got.ConnectRetryDelay)
				assert.Equal(t, 1*time.Second, got.ConnectTimeout)
				assert.NotNil(t, got.OnConnectionUp)
				assert.NotNil(t, got.OnConnectError)
				assert.NotNil(t, got.ClientConfig.OnServerDisconnect)
				assert.NotNil(t, got.ClientConfig.OnClientError)
				assert.NotNil(t, got.ClientConfig.PublishHook)
			}
		})
	}
}
