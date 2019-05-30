package info

import (
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/errors"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

type info struct {
	Name       string                   `json:"name,omitempty"`
	Version    string                   `json:"version,omitempty"`
	Host       string                   `json:"host,omitempty"`
	Configs    map[string]interface{}   `json:"configs,omitempty"`
	Components []map[string]interface{} `json:"components,omitempty"`
	Metrics    map[string]string        `json:"metrics,omitempty"`
	Doc        string                   `json:"doc,omitempty"`
}

var (
	// ServiceInfo holds the information of the service.
	serviceInfo = info{
		Configs: make(map[string]interface{}),
		Metrics: make(map[string]string),
	}
	mu = sync.Mutex{}
)

// Marshal returns the service info as a byte slice.
func Marshal() ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	return json.Encode(serviceInfo)
}

// UpdateName to the info.
func UpdateName(n string) {
	mu.Lock()
	defer mu.Unlock()
	serviceInfo.Name = n
}

// UpdateVersion to the info.
func UpdateVersion(v string) {
	mu.Lock()
	defer mu.Unlock()
	serviceInfo.Version = v
}

// UpdateHost to the info.
func UpdateHost(h string) {
	mu.Lock()
	defer mu.Unlock()
	serviceInfo.Host = h
}

// UpsertMetric to the info.
func UpsertMetric(n, d, typ string) {
	mu.Lock()
	defer mu.Unlock()
	serviceInfo.Metrics[n] = fmt.Sprintf("[%s] %s", typ, d)
}

// ImportDoc adds documentation from a markdown file.
func ImportDoc(file string) error {
	mu.Lock()
	defer mu.Unlock()
	if file == "" {
		serviceInfo.Doc = ""
		return errors.New("no file provided")
	}
	cnt, err := ioutil.ReadFile(file)
	if err != nil {
		serviceInfo.Doc = ""
		return errors.Errorf("failed to read file %s", file)
	}
	serviceInfo.Doc = string(blackfriday.Run(cnt, blackfriday.WithExtensions(blackfriday.CommonExtensions)))
	return nil
}

// UpsertConfig to the info.
func UpsertConfig(n string, v interface{}) {
	mu.Lock()
	defer mu.Unlock()
	serviceInfo.Configs[n] = v
}

// AppendComponent to the info.
func AppendComponent(i map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()
	serviceInfo.Components = append(serviceInfo.Components, i)
}
