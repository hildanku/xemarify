package apiclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/model"
)

func NewHTTPClient(insecure bool) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}

	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
}

func Register(client *http.Client, endpoint, enrollmentToken string, payload model.RegisterRequest) (*model.RegisterResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := JoinURL(endpoint, "/api/v1/agents/register")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(model.EnrollmentTokenHeader, enrollmentToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("register failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var out model.RegisterResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}

	if strings.TrimSpace(out.AgentID) == "" || strings.TrimSpace(out.AgentSecret) == "" {
		return nil, fmt.Errorf("register returned empty credentials")
	}

	return &out, nil
}

func PostJSON(ctx context.Context, client *http.Client, url string, agentSecret string, payload any, expectedStatus int) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(model.AgentSecretHeader, agentSecret)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func JoinURL(endpoint, path string) string {
	return strings.TrimRight(endpoint, "/") + path
}
