package openaichat

import (
	"fmt"
	"net/http"
)

type customTransport struct {
	http.RoundTripper

	BearerToken string
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.BearerToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// This is for peventing the SSE client from infinitely retrying requests that end up with 400 Bad Request.
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", req.Response.StatusCode)
	}

	return resp, err
}
