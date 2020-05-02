package seventh

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	clienthttp "github.com/beatlabs/patron/client/http"
)

func DoTimingRequest(ctx context.Context) (string, error) {
	request, err := http.NewRequest("GET", "http://localhost:50006/", nil)
	if err != nil {
		return "", fmt.Errorf("failed create route request: %w", err)
	}
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		return "", fmt.Errorf("could not create http client: %w", err)
	}

	response, err := cl.Do(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed create get to seventh service: %w", err)
	}

	tb, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode timing response body: %w", err)
	}

	var rgx = regexp.MustCompile(`\((.*?)\)`)
	timeInstance := rgx.FindStringSubmatch(string(tb))
	if len(timeInstance) == 1 {
		return "", fmt.Errorf("could not match timeinstance from response %s", string(tb))
	}
	return timeInstance[1], nil
}
