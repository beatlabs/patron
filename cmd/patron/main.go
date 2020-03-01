package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

type genData struct {
	Name   string
	Module string
	Path   string
	Vendor bool
}

func main() {
	module := flag.String("m", "", `define the module name ("github.com/beatlabs/patron")`)
	path := flag.String("p", "", "define the project folder (defaults to current)")
	vendor := flag.Bool("d", true, "define vendoring behavior (default true)")
	flag.Parse()

	gd, err := getGenData(*path, *module, *vendor)
	if err != nil {
		fmt.Printf("error occurred. %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	err = createPathAndChdir(gd)
	if err != nil {
		log.Fatalf("failed to create path: %v", err)
	}

	err = setupGit()
	if err != nil {
		log.Fatalf("failed to create git: %v", err)
	}

	err = createMain(gd)
	if err != nil {
		log.Fatalf("failed to create main: %v", err)
	}

	err = createGitIgnore()
	if err != nil {
		log.Fatalf("failed to create .gitignore: %v", err)
	}

	err = createDockerfile(gd)
	if err != nil {
		log.Fatalf("failed to create Dockerfile: %v", err)
	}

	err = createReadme(gd)
	if err != nil {
		log.Fatalf("failed to create README.md: %v", err)
	}

	err = goMod(gd)
	if err != nil {
		log.Fatalf("failed to initialize go mod support: %v", err)
	}

	err = goFormat()
	if err != nil {
		log.Fatalf("failed to execute go fmt: %v", err)
	}

	err = gitCommit()
	if err != nil {
		log.Fatalf("failed to commit initially to git: %v", err)
	}
	log.Print("completed successful")
}

func getGenData(path, module string, vendor bool) (*genData, error) {
	if path == "" {
		return nil, errors.New("path is required")
	}

	if module == "" {
		return nil, errors.New("module is required")
	}

	return &genData{
		Path:   path,
		Vendor: vendor,
		Module: module,
		Name:   nameFromModule(module),
	}, nil
}

func nameFromModule(module string) string {
	lst := strings.LastIndex(module, "/")
	if lst == -1 {
		return module
	}

	return module[lst+1:]
}

func setupGit() error {
	log.Printf("creating git repository")
	return exec.Command("git", "init").Run()
}

func createGitIgnore() error {
	log.Printf("copying .gitignore")
	return ioutil.WriteFile("README.md", gitIgnoreContent(), 0664)
}

func createDockerfile(gd *genData) error {
	buf, err := dockerfileContent(gd)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("Dockerfile", buf, 0664)
}

func createReadme(gd *genData) error {
	return ioutil.WriteFile("README.md", readmeContent(gd), 0664)
}

func goMod(gd *genData) error {
	log.Print("setup go module support")
	out, err := exec.Command("go", "mod", "init", gd.Module).CombinedOutput()
	log.Print(string(out))
	if err != nil {
		return errors.New(string(out))
	}
	log.Print("go mod tidy")
	out, err = exec.Command("go", "mod", "tidy").CombinedOutput()
	log.Print(string(out))
	if err != nil {
		return errors.New(string(out))
	}
	if gd.Vendor {
		log.Print("add vendoring")
		out, err := exec.Command("go", "mod", "vendor").CombinedOutput()
		log.Print(string(out))
		if err != nil {
			return errors.New(string(out))
		}
	}
	return nil
}

func createPathAndChdir(gd *genData) error {
	log.Printf("create folder: %s", gd.Path)
	err := os.MkdirAll(gd.Path, 0775)
	if err != nil {
		return err
	}

	log.Printf("cd into: %s", gd.Path)
	return os.Chdir(gd.Path)
}

func createMain(gd *genData) error {
	folder := fmt.Sprintf("cmd/%s", gd.Name)
	log.Printf("create folder: %s", folder)
	err := os.MkdirAll(folder, 0775)
	if err != nil {
		return err
	}

	file := fmt.Sprintf("%s/main.go", folder)
	log.Printf("create file: %s", file)
	buf, err := mainContent(gd)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, buf, 0664)
}

func gitCommit() error {
	log.Printf("git: add .")
	err := exec.Command("git", "add", ".").Run()
	if err != nil {
		return err
	}
	log.Printf("git: commit")
	return exec.Command("git", "commit", "-m", "Initial commit").Run()
}

func goFormat() error {
	log.Print("go: fmt ./...")
	out, err := exec.Command("go", "fmt", "./...").CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}
	return nil
}

func dockerfileContent(gd *genData) ([]byte, error) {
	cnt := `FROM golang:latest as builder
RUN cd ..
RUN mkdir {{ .Name}}
WORKDIR {{ .Name}}
COPY . ./
ARG version=dev
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -ldflags "-X main.version=$version" -o {{ .Name}} ./cmd/{{ .Name}}/main.go 

FROM scratch
COPY --from=builder /go/{{ .Name}}/{{ .Name}} .
CMD ["./{{ .Name}}"]
`
	t := template.Must(template.New("docker").Parse(cnt))
	b := new(bytes.Buffer)
	err := t.ExecuteTemplate(b, "docker", gd)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func mainContent(gd *genData) ([]byte, error) {
	cnt := `package main

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
	name := "{{ .Name}}"

	err := patron.Setup(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	srv, err := patron.New(name, version)
	if err != nil {
		log.Fatalf("failed to create service %v", err)
	}

	ctx := context.Background()
	err = srv.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %v", err)
	}
}
`
	t := template.Must(template.New("main").Parse(cnt))
	b := new(bytes.Buffer)
	err := t.ExecuteTemplate(b, "main", gd)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func readmeContent(gd *genData) []byte {
	return []byte(fmt.Sprintf("# %s", gd.Name))
}

func gitIgnoreContent() []byte {
	content := `# Binaries for programs and plugins
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
	return []byte(content)
}
