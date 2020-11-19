package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getGenData(t *testing.T) {
	type args struct {
		path   string
		module string
		vendor bool
	}
	tests := map[string]struct {
		args   args
		want   *genData
		expErr string
	}{
		"success": {
			args: args{path: "path", module: "module", vendor: true},
			want: &genData{Name: "module", Module: "module", Path: "path", Vendor: true},
		},
		"success with complex module": {
			args: args{path: "path", module: "github.com/module", vendor: true},
			want: &genData{Name: "module", Module: "github.com/module", Path: "path", Vendor: true},
		},
		"missing path": {
			args:   args{path: "", module: "module", vendor: true},
			expErr: "path is required",
		},
		"missing module": {
			args:   args{path: "path", module: "", vendor: true},
			expErr: "module is required",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := getGenData(tt.args.path, tt.args.module, tt.args.vendor)
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_dockerfileContent(t *testing.T) {
	expected := `FROM golang:latest as builder
RUN cd ..
RUN mkdir name
WORKDIR name
COPY . ./
ARG version=dev
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -ldflags "-X main.version=$version" -o name ./cmd/name/main.go 

FROM scratch
COPY --from=builder /go/name/name .
CMD ["./name"]
`
	gd := &genData{Name: "name", Module: "module", Path: "path", Vendor: true}
	got, err := dockerfileContent(gd)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(got))
}

func Test_mainContent(t *testing.T) {
	expected := `package main

import (
	"context"
	"fmt"
	"os"
	
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/log"
)

var (
	version = "dev"
)

func main() {
	name := "name"

	service, err := patron.New(name, version, patron.TextLogger())
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	ctx := context.Background()
	err = service.Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}
`
	gd := &genData{Name: "name", Module: "module", Path: "path", Vendor: true}
	got, err := mainContent(gd)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(got))
}

func Test_gitIgnoreContent(t *testing.T) {
	expected := `# Binaries for programs and plugins
*.exe
*.dll
*.so
*.DS_Store
*.dylib
debug/

# JetBrains
.idea/*
*.iws
out/
.idea_modules/

# JIRA plugin
atlassian-ide-plugin.xml

# VS Code
*.vscode
.vscode/*

# Test binary, build with "go test -c"
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Project-local glide cache, RE: https://github.com/Masterminds/glide/issues/736
.glide/
`
	got := gitIgnoreContent()
	assert.Equal(t, expected, string(got))
}

func Test_readmeContent(t *testing.T) {
	expected := "# name"

	gd := &genData{Name: "name", Module: "module", Path: "path", Vendor: true}
	got := readmeContent(gd)
	assert.Equal(t, expected, string(got))
}
