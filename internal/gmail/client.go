package gmail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

const gmailAPIBaseURL = "https://gmail.googleapis.com/gmail/v1/users/me"

// httpClient creates an HTTP client with OAuth authentication for the given account.
func (a *GmailAdapter) httpClient(ctx context.Context, accountID string) (*http.Client, error) {
	token, err := a.tokenStore.Load(ctx, accountID, a.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	tokenSource := a.oauthCfg.TokenSource(ctx, token)

	return oauth2.NewClient(ctx, tokenSource), nil
}

// gmailGet makes a GET request to the Gmail API.
func gmailGet(client *http.Client, path string, result interface{}) error {
	url := gmailAPIBaseURL + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gmail API error: %d - %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// gmailPost makes a POST request to the Gmail API.
func gmailPost(client *http.Client, path string, body interface{}, result interface{}) error {
	url := gmailAPIBaseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gmail API error: %d - %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// gmailDelete makes a DELETE request to the Gmail API.
func gmailDelete(client *http.Client, path string) error {
	url := gmailAPIBaseURL + path

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gmail API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}
