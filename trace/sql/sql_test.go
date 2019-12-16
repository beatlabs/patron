package sql

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDSN(t *testing.T) {
	tests := map[string]struct {
		dsn  string
		want DSNInfo
	}{
		"generic case":          {"username:password@protocol(address)/dbname?param=value", DSNInfo{"", "dbname", "address", "username", "protocol"}},
		"empty DSN":             {"/", DSNInfo{"", "", "", "", ""}},
		"dbname only":           {"/dbname", DSNInfo{"", "dbname", "", "", ""}},
		"multiple @":            {"user:p@/ssword@/", DSNInfo{"", "", "", "user", ""}},
		"driver and multiple @": {"postgresql://user:p@/ssword@/", DSNInfo{"postgresql://", "", "", "user", ""}},
		"unix socket":           {"user@unix(/path/to/socket)/dbname?charset=utf8", DSNInfo{"", "dbname", "/path/to/socket", "user", "unix"}},
		"params added":          {"user:password@/dbname?param1=val1&param2=val2&param3=val3", DSNInfo{"", "dbname", "", "user", ""}},
		"IP as address":         {"bruce:hunter2@tcp(127.0.0.1)/arkhamdb?param=value", DSNInfo{"", "arkhamdb", "127.0.0.1", "bruce", "tcp"}},
		"@ in path to socker":   {"user@unix(/path/to/mydir@/socket)/dbname?charset=utf8", DSNInfo{"", "dbname", "/path/to/mydir@/socket", "user", "unix"}},
		"port in address":       {"user:password@tcp(localhost:5555)/dbname?charset=utf8&tls=true", DSNInfo{"", "dbname", "localhost:5555", "user", "tcp"}},
		"multiple ':'":          {"us:er:name:password@memory(localhost:5555)/dbname?charset=utf8&tls=true", DSNInfo{"", "dbname", "localhost:5555", "us", "memory"}},
		"IPv6 provided":         {"user:p@ss(word)@tcp([c023:9350:225b:671a:2cdd:3d83:7c19:ca42]:80)/dbname?loc=Local", DSNInfo{"", "dbname", "[c023:9350:225b:671a:2cdd:3d83:7c19:ca42]:80", "user", "tcp"}},
		"empty string":          {"", DSNInfo{"", "", "", "", ""}},
		"non-matching string":   {"rosebud", DSNInfo{"", "", "", "", ""}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseDSN(tc.dsn)
			assert.Equal(t, got, tc.want)
		})
	}
}
