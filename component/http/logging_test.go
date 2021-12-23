package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const complexConfig = "200;(210,212);(220,222];[230,232);[240,242]"

func TestStatusCode(t *testing.T) {
	t.Parallel()
	type args struct {
		cfg        string
		statusCode int
	}
	tests := map[string]struct {
		args               args
		expectedResult     bool
		expectedParsingErr bool
	}{
		"empty":                                    {args: args{cfg: "", statusCode: 400}, expectedResult: false},
		"single element - true":                    {args: args{cfg: "400", statusCode: 400}, expectedResult: true},
		"single element - false":                   {args: args{cfg: "400", statusCode: 401}, expectedResult: false},
		"complex config - single element - true":   {args: args{cfg: complexConfig, statusCode: 200}, expectedResult: true},
		"complex config - single element - false":  {args: args{cfg: complexConfig, statusCode: 300}, expectedResult: false},
		"complex config - excl/excl range - true":  {args: args{cfg: complexConfig, statusCode: 211}, expectedResult: true},
		"complex config - excl/excl range - false": {args: args{cfg: complexConfig, statusCode: 212}, expectedResult: false},
		"complex config - excl/incl range - true":  {args: args{cfg: complexConfig, statusCode: 222}, expectedResult: true},
		"complex config - excl/incl range - false": {args: args{cfg: complexConfig, statusCode: 220}, expectedResult: false},
		"complex config - incl/excl range - true":  {args: args{cfg: complexConfig, statusCode: 230}, expectedResult: true},
		"complex config - incl/excl range - false": {args: args{cfg: complexConfig, statusCode: 232}, expectedResult: false},
		"complex config - incl/incl range - true":  {args: args{cfg: complexConfig, statusCode: 240}, expectedResult: true},
		"complex config - incl/incl range - false": {args: args{cfg: complexConfig, statusCode: 243}, expectedResult: false},
		"config error - missing code range":        {args: args{cfg: "[200,]"}, expectedParsingErr: true},
		"config error - invalid start interval":    {args: args{cfg: "x200,201]"}, expectedParsingErr: true},
		"config error - invalid end interval":      {args: args{cfg: "[200,201x"}, expectedParsingErr: true},
		"config error - invalid start":             {args: args{cfg: "[2x0,201]"}, expectedParsingErr: true},
		"config error - invalid end":               {args: args{cfg: "[200,2x1]"}, expectedParsingErr: true},
		"config error - invalid range":             {args: args{cfg: "[200,201,202]"}, expectedParsingErr: true},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h, err := newStatusCodeLoggerHandler(tt.args.cfg)
			if tt.expectedParsingErr {
				assert.Error(t, err)
			} else {
				got := h.shouldLog(tt.args.statusCode)
				assert.Equal(t, tt.expectedResult, got)
			}
		})
	}
}

func BenchmarkName(b *testing.B) {
	h, err := newStatusCodeLoggerHandler(complexConfig)
	assert.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = h.shouldLog(403)
	}
}
