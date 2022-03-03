package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"
)

// this program is meant to be used to check the whole 'examples' cluster. It's meant to be put in the CI
// so that new developments do not break the examples. We can see this as a single E2E test check
// the program is supposed to start a transaction that flows through all the services under the 'examples'
// folder and then check that the transaction actually triggered all the services in the chain until the
// 'leaves'
func main() {
	// create simple client with timeout set to 1 second
	client := http.Client{
		Timeout: 1 * time.Second,
	}

	// perform a client call that is supposed to trigger a call across all the services
	jsonStr := []byte(`{"firstname":"John","lastname":"Doe"}`)
	req, err := http.NewRequest("POST", "http://localhost:50000/api", bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatalf("Response supposed to be status code %d instead it is %d (body: %q)",
			http.StatusCreated, resp.StatusCode, string(body))
	}

	// wait 1 second so that metrics can be flushed to http endpoint
	time.Sleep(1 * time.Second)

	// services to be checked
	svcsToUrls := map[string]string{
		"http-svc":       "http://localhost:50000/",
		"http-sec-svc":   "http://localhost:50001/",
		"http-cache-svc": "http://localhost:50007/",
		"kafka-svc":      "http://localhost:50002/",
		"amqp-svc":       "http://localhost:50003/",
		"sqs-svc":        "http://localhost:50004/",
		"grpc-svc":       "http://localhost:50005/",
	}

	// collect the metrics for each of the services
	// and early terminate in case of error
	responses := make(map[string]*http.Response)
	for svc, url := range svcsToUrls {
		urlMetrics := url + "metrics"
		log.Printf("Checking service %s [url %s]", svc, urlMetrics)
		rsp, err := client.Get(urlMetrics)
		if err != nil {
			log.Fatalf("Svc %s is not reachable. HTTP error: ", err)
		}
		if rsp == nil {
			log.Fatalf("Svc %s [url %s] lead to no response", svc, urlMetrics)
		}
		if rsp.StatusCode != http.StatusOK {
			log.Fatalf("Svc %s [url %s] lead to HTTP status code : %d", svc, urlMetrics, rsp.StatusCode)
		}
		responses[svc] = rsp
	}

	// To check that the message flowed across all the services we check the leaves of the message flow.
	// The leaves are 'grpc-svc' and 'http-cache-svc'
	// To check that the message made through them, we check that the metrics endpoint contain certain strings
	// by using a regex

	// services to url map
	svcsToRegex := map[string]string{
		"http-cache-svc": `client_redis_cmd_duration_seconds_count{.*,success="true"} [1-9][0-9]*`,
		"grpc-svc":       `component_grpc_handled_total{.*grpc_code="OK".*} [1-9][0-9]*`,
	}

	for svc, re := range svcsToRegex {
		if !containsRegex(responses[svc], re) {
			log.Fatalf("Svc %s does not contain the regular expression %s in the metrics", svc, re)
		}
	}

	fmt.Printf("Successful E2E test\n")
}

func containsRegex(resp *http.Response, re string) bool {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	str := string(bodyBytes)
	ok, err := regexp.MatchString(re, str)
	if err != nil {
		log.Fatal(err)
	}
	return ok
}
