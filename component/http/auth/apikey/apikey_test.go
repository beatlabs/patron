package apikey

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockValidator struct {
	err     error
	success bool
}

func (mv MockValidator) Validate(key string) (bool, error) {
	if mv.err != nil {
		return false, mv.err
	}
	return mv.success, nil
}

func TestNew(t *testing.T) {
	type args struct {
		val Validator
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{val: &MockValidator{}}, wantErr: false},
		{name: "failed due to nil validator", args: args{val: nil}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.val)
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

func TestAuthenticator_Authenticate(t *testing.T) {
	reqOk, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)
	reqOk.Header.Set("Authorization", "Apikey 123456")
	reqMissingHeader, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)
	reqMissingKey, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)
	reqMissingKey.Header.Set("Authorization", "Apikey")
	reqInvalidAuthMethod, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)
	reqInvalidAuthMethod.Header.Set("Authorization", "Bearer 123456")

	type fields struct {
		val Validator
	}
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{name: "authenticated", fields: fields{val: &MockValidator{success: true}}, args: args{req: reqOk}, want: true, wantErr: false},
		{name: "not authenticated, validation failed", fields: fields{val: &MockValidator{success: false}}, args: args{req: reqOk}, want: false, wantErr: false},
		{name: "failed, validation returned err", fields: fields{val: &MockValidator{err: errors.New("TEST")}}, args: args{req: reqOk}, want: false, wantErr: true},
		{name: "not authenticated, header missing", fields: fields{val: &MockValidator{success: false}}, args: args{req: reqMissingHeader}, want: false, wantErr: false},
		{name: "not authenticated, missing key", fields: fields{val: &MockValidator{success: false}}, args: args{req: reqMissingKey}, want: false, wantErr: false},
		{name: "not authenticated, invalid auth method", fields: fields{val: &MockValidator{success: false}}, args: args{req: reqInvalidAuthMethod}, want: false, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Authenticator{
				val: tt.fields.val,
			}
			got, err := a.Authenticate(tt.args.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
