package main

import (
	"fmt"
	"io"
	"net/http"
)

func (a *app) httpRequest(url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could create get request for %s: %w", url, err)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status code for %s not 200: %d %s", url, resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body from %s: %w", url, err)
	}

	return body, nil
}
