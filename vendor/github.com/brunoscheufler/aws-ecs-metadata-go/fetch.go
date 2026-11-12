package metadata

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
)

func fetch(ctx context.Context, client *http.Client, metadataUrl string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, metadataUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create metadata request: %w", err)
	}
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send metadata request: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read metadata response: %w", err)
	}

	return body, nil
}
