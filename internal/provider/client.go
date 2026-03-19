package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

var ErrNotFound = errors.New("resource not found")

type Client struct {
	baseURL    *url.URL
	apiToken   string
	userAgent  string
	httpClient *http.Client
}

func NewClient(rawBaseURL, apiToken, version string) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	parsed.Path = strings.TrimSuffix(parsed.Path, "/")

	userAgent := "terraform-provider-tensordock"
	if strings.TrimSpace(version) != "" {
		userAgent = userAgent + "/" + strings.TrimSpace(version)
	}

	return &Client{
		baseURL:   parsed,
		apiToken:  apiToken,
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

type CreateInstanceInput struct {
	Name           string
	Image          string
	LocationID     string
	VCPUCount      int64
	RAMGB          int64
	StorageGB      int64
	GPUType        string
	GPUCount       int64
	UseDedicatedIP bool
	SSHPublicKey   string
	CloudInit      map[string]any
}

type ModifyInstanceInput struct {
	VCPUCount int64
	RAMGB     int64
	StorageGB int64
	GPUType   string
	GPUCount  int64
}

type Instance struct {
	ID           string
	Name         string
	Status       string
	IPAddress    string
	RateHourly   *float64
	VCPUCount    int64
	RAMGB        int64
	StorageGB    int64
	GPUType      string
	GPUCount     int64
	PortForwards []PortForward
}

type PortForward struct {
	InternalPort int64 `json:"internal_port"`
	ExternalPort int64 `json:"external_port"`
}

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("TensorDock API returned HTTP %d: %s", e.StatusCode, e.Body)
}

type gpuDetails struct {
	Count  int64  `json:"count"`
	V0Name string `json:"v0Name"`
}

type rawInstance struct {
	ID           string                     `json:"id"`
	Name         string                     `json:"name"`
	Status       string                     `json:"status"`
	IPAddress    string                     `json:"ipAddress"`
	PortForwards []PortForward              `json:"portForwards"`
	Resources    map[string]json.RawMessage `json:"resources"`
	RateHourly   *float64                   `json:"rateHourly"`
}

func (c *Client) CreateInstance(ctx context.Context, input CreateInstanceInput) (Instance, error) {
	payload := map[string]any{
		"data": map[string]any{
			"type": "virtualmachine",
			"attributes": map[string]any{
				"name":  input.Name,
				"type":  "virtualmachine",
				"image": input.Image,
				"resources": map[string]any{
					"vcpu_count": input.VCPUCount,
					"ram_gb":     input.RAMGB,
					"storage_gb": input.StorageGB,
					"gpus": map[string]any{
						input.GPUType: map[string]any{
							"count": input.GPUCount,
						},
					},
				},
				"location_id":    input.LocationID,
				"useDedicatedIp": input.UseDedicatedIP,
				"ssh_key":        input.SSHPublicKey,
			},
		},
	}

	if len(input.CloudInit) > 0 {
		payload["data"].(map[string]any)["attributes"].(map[string]any)["cloud_init"] = input.CloudInit
	}

	body, err := c.doJSON(ctx, http.MethodPost, "/instances", payload)
	if err != nil {
		return Instance{}, err
	}

	var resp struct {
		Data struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return Instance{}, fmt.Errorf("decode create instance response: %w", err)
	}

	if resp.Data.ID == "" {
		return Instance{}, fmt.Errorf("decode create instance response: missing instance ID")
	}

	return Instance{
		ID:     resp.Data.ID,
		Name:   resp.Data.Name,
		Status: resp.Data.Status,
	}, nil
}

func (c *Client) GetInstance(ctx context.Context, id string) (Instance, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances/"+id, nil)
	if err != nil {
		return Instance{}, err
	}

	instance, err := decodeInstance(body)
	if err != nil {
		return Instance{}, err
	}

	return instance, nil
}

func (c *Client) StartInstance(ctx context.Context, id string) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/instances/"+id+"/start", nil)
	return err
}

func (c *Client) StopInstance(ctx context.Context, id string) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/instances/"+id+"/stop", nil)
	return err
}

func (c *Client) DeleteInstance(ctx context.Context, id string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/instances/"+id, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}

	return err
}

func (c *Client) ModifyInstance(ctx context.Context, id string, input ModifyInstanceInput) error {
	payload := map[string]any{
		"cpuCores": input.VCPUCount,
		"ramGb":    input.RAMGB,
		"diskGb":   input.StorageGB,
		"gpus": map[string]any{
			"gpuV0Name": input.GPUType,
			"count":     input.GPUCount,
		},
	}

	_, err := c.doJSON(ctx, http.MethodPut, "/instances/"+id+"/modify", payload)
	return err
}

func (c *Client) WaitForStatus(ctx context.Context, id string, targets ...string) (Instance, error) {
	normalizedTargets := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		normalizedTargets[normalizeStatus(target)] = struct{}{}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		instance, err := c.GetInstance(ctx, id)
		if err == nil {
			if _, ok := normalizedTargets[normalizeStatus(instance.Status)]; ok {
				return instance, nil
			}
		} else if errors.Is(err, ErrNotFound) {
			return Instance{}, err
		}

		select {
		case <-ctx.Done():
			return Instance{}, fmt.Errorf("wait for instance %s to reach %v: %w", id, targets, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) WaitForDeletion(ctx context.Context, id string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		_, err := c.GetInstance(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for instance %s deletion: %w", id, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, payload any) ([]byte, error) {
	endpointURL := *c.baseURL
	endpointURL.Path = path.Join(endpointURL.Path, strings.TrimPrefix(endpoint, "/"))

	var requestBody io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpointURL.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("User-Agent", c.userAgent)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &apiError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	return body, nil
}

func decodeInstance(body []byte) (Instance, error) {
	var raw rawInstance
	if err := json.Unmarshal(body, &raw); err == nil && raw.ID != "" {
		return convertRawInstance(raw)
	}

	var wrapped struct {
		Data rawInstance `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Data.ID != "" {
		return convertRawInstance(wrapped.Data)
	}

	return Instance{}, fmt.Errorf("decode instance response: unsupported response shape")
}

func convertRawInstance(raw rawInstance) (Instance, error) {
	instance := Instance{
		ID:           raw.ID,
		Name:         raw.Name,
		Status:       raw.Status,
		IPAddress:    raw.IPAddress,
		PortForwards: raw.PortForwards,
		RateHourly:   raw.RateHourly,
	}

	if len(raw.Resources) > 0 {
		if v, ok := raw.Resources["vcpu_count"]; ok {
			if err := json.Unmarshal(v, &instance.VCPUCount); err != nil {
				return Instance{}, fmt.Errorf("decode resources.vcpu_count: %w", err)
			}
		}
		if v, ok := raw.Resources["ram_gb"]; ok {
			if err := json.Unmarshal(v, &instance.RAMGB); err != nil {
				return Instance{}, fmt.Errorf("decode resources.ram_gb: %w", err)
			}
		}
		if v, ok := raw.Resources["storage_gb"]; ok {
			if err := json.Unmarshal(v, &instance.StorageGB); err != nil {
				return Instance{}, fmt.Errorf("decode resources.storage_gb: %w", err)
			}
		}
		if v, ok := raw.Resources["gpus"]; ok {
			gpuMap := map[string]gpuDetails{}
			if err := json.Unmarshal(v, &gpuMap); err != nil {
				return Instance{}, fmt.Errorf("decode resources.gpus: %w", err)
			}
			instance.GPUType, instance.GPUCount = flattenGPUMap(gpuMap)
		}
	}

	return instance, nil
}

func flattenGPUMap(gpus map[string]gpuDetails) (string, int64) {
	if len(gpus) == 0 {
		return "", 0
	}

	keys := make([]string, 0, len(gpus))
	for key := range gpus {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	first := gpus[keys[0]]
	if first.V0Name != "" {
		return first.V0Name, first.Count
	}

	return keys[0], first.Count
}

func normalizeStatus(status string) string {
	normalized := strings.TrimSpace(strings.ToLower(status))
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}

func normalizePowerState(status string) string {
	switch normalizeStatus(status) {
	case "running", "starting":
		return "running"
	case "stopped", "stopping", "stoppeddisassociated":
		return "stopped"
	default:
		trimmed := strings.TrimSpace(strings.ToLower(status))
		if trimmed == "" {
			return ""
		}
		return trimmed
	}
}
