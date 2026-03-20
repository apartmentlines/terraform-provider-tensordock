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
		userAgent += "/" + strings.TrimSpace(version)
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
	HostnodeID     string
	VCPUCount      int64
	RAMGB          int64
	StorageGB      int64
	GPUType        string
	GPUCount       int64
	UseDedicatedIP bool
	PortForwards   []PortForward
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

type SecretSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Secret struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
}

type ResourceLimits struct {
	MaxVCPUs     int64 `json:"max_vcpus"`
	MaxRAMGB     int64 `json:"max_ram_gb"`
	MaxStorageGB int64 `json:"max_storage_gb"`
}

type Pricing struct {
	PerVCPUHR      float64 `json:"per_vcpu_hr"`
	PerGBRAMHR     float64 `json:"per_gb_ram_hr"`
	PerGBStorageHR float64 `json:"per_gb_storage_hr"`
}

type NetworkFeatures struct {
	DedicatedIPAvailable   bool `json:"dedicated_ip_available"`
	PortForwardingAvailble bool `json:"port_forwarding_available"`
	NetworkStorageAvailble bool `json:"network_storage_available"`
}

type LocationGPU struct {
	V0Name      string          `json:"v0Name"`
	DisplayName string          `json:"displayName"`
	MaxCount    int64           `json:"max_count"`
	PricePerHR  float64         `json:"price_per_hr"`
	Resources   ResourceLimits  `json:"resources"`
	Pricing     Pricing         `json:"pricing"`
	Network     NetworkFeatures `json:"network_features"`
}

type Location struct {
	ID            string        `json:"id"`
	City          string        `json:"city"`
	StateProvince string        `json:"stateprovince"`
	Country       string        `json:"country"`
	Tier          int64         `json:"tier"`
	GPUs          []LocationGPU `json:"gpus"`
}

type HostnodeGPU struct {
	V0Name         string  `json:"v0Name"`
	AvailableCount int64   `json:"availableCount"`
	PricePerHR     float64 `json:"price_per_hr"`
}

type HostnodeLocation struct {
	UUID                   string `json:"uuid"`
	City                   string `json:"city"`
	StateProvince          string `json:"stateprovince"`
	Country                string `json:"country"`
	HasNetworkStorage      bool   `json:"has_network_storage"`
	NetworkSpeedGbps       int64  `json:"network_speed_gbps"`
	NetworkSpeedUploadGbps int64  `json:"network_speed_upload_gbps"`
	Organization           string `json:"organization"`
	OrganizationName       string `json:"organizationName"`
	Tier                   int64  `json:"tier"`
}

type HostnodeAvailableResources struct {
	GPUs                 []HostnodeGPU `json:"gpus"`
	VCPUCount            int64         `json:"vcpu_count,omitempty"`
	RAMGB                int64         `json:"ram_gb,omitempty"`
	StorageGB            int64         `json:"storage_gb,omitempty"`
	MaxVCPUsPerGPU       int64         `json:"max_vcpus_per_gpu,omitempty"`
	MaxRAMPerGPU         int64         `json:"max_ram_per_gpu,omitempty"`
	MaxVCPUs             int64         `json:"max_vcpus,omitempty"`
	MaxRAMGB             int64         `json:"max_ram_gb,omitempty"`
	MaxStorageGB         int64         `json:"max_storage_gb,omitempty"`
	AvailablePorts       []int64       `json:"available_ports,omitempty"`
	HasPublicIPAvailable bool          `json:"has_public_ip_available"`
}

type Hostnode struct {
	ID                 string                     `json:"id"`
	LocationID         string                     `json:"location_id"`
	Engine             string                     `json:"engine"`
	UptimePercentage   float64                    `json:"uptime_percentage"`
	AvailableResources HostnodeAvailableResources `json:"available_resources"`
	Pricing            Pricing                    `json:"pricing"`
	Location           HostnodeLocation           `json:"location"`
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

func (c *Client) ListSecrets(ctx context.Context) ([]SecretSummary, error) {
	var resp struct {
		Data struct {
			Secrets []SecretSummary `json:"secrets"`
		} `json:"data"`
	}

	body, err := c.doJSON(ctx, http.MethodGet, "/secrets", nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode secrets response: %w", err)
	}

	return resp.Data.Secrets, nil
}

func (c *Client) CreateSecret(ctx context.Context, name, secretType, value string) (Secret, error) {
	payload := map[string]any{
		"data": map[string]any{
			"type": "secret",
			"attributes": map[string]any{
				"name":  name,
				"type":  secretType,
				"value": value,
			},
		},
	}

	body, err := c.doJSON(ctx, http.MethodPost, "/secrets", payload)
	if err != nil {
		return Secret{}, err
	}

	var resp struct {
		Data Secret `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Secret{}, fmt.Errorf("decode create secret response: %w", err)
	}
	if resp.Data.ID == "" {
		return Secret{}, fmt.Errorf("decode create secret response: missing secret ID")
	}

	return resp.Data, nil
}

func (c *Client) GetSecret(ctx context.Context, id string) (Secret, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/secrets/"+id, nil)
	if err != nil {
		return Secret{}, err
	}

	var resp struct {
		Data Secret `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Secret{}, fmt.Errorf("decode secret response: %w", err)
	}
	if resp.Data.ID == "" {
		return Secret{}, fmt.Errorf("decode secret response: missing secret ID")
	}

	return resp.Data, nil
}

func (c *Client) DeleteSecret(ctx context.Context, id string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/secrets/"+id, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}

	return err
}

func (c *Client) ListLocations(ctx context.Context) ([]Location, error) {
	var resp struct {
		Data struct {
			Locations []Location `json:"locations"`
		} `json:"data"`
	}

	body, err := c.doJSON(ctx, http.MethodGet, "/locations", nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode locations response: %w", err)
	}

	return resp.Data.Locations, nil
}

func (c *Client) ListHostnodes(ctx context.Context) ([]Hostnode, error) {
	var resp struct {
		Data struct {
			Hostnodes []Hostnode `json:"hostnodes"`
		} `json:"data"`
	}

	body, err := c.doJSON(ctx, http.MethodGet, "/hostnodes", nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode hostnodes response: %w", err)
	}

	return resp.Data.Hostnodes, nil
}

func (c *Client) GetHostnode(ctx context.Context, id string) (Hostnode, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/hostnodes/"+id, nil)
	if err != nil {
		return Hostnode{}, err
	}

	var resp struct {
		Data Hostnode `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Hostnode{}, fmt.Errorf("decode hostnode response: %w", err)
	}
	if resp.Data.ID == "" {
		return Hostnode{}, fmt.Errorf("decode hostnode response: missing hostnode ID")
	}

	return resp.Data, nil
}

func (c *Client) CreateInstance(ctx context.Context, input CreateInstanceInput) (Instance, error) {
	attributes := map[string]any{
		"name":  input.Name,
		"type":  "virtualmachine",
		"image": input.Image,
		"resources": map[string]any{
			"vcpu_count": input.VCPUCount,
			"ram_gb":     input.RAMGB,
			"storage_gb": input.StorageGB,
		},
	}

	if input.LocationID != "" {
		attributes["location_id"] = input.LocationID
	}
	if input.HostnodeID != "" {
		attributes["hostnode_id"] = input.HostnodeID
	}
	if input.UseDedicatedIP {
		attributes["useDedicatedIp"] = true
	}
	if input.SSHPublicKey != "" {
		attributes["ssh_key"] = input.SSHPublicKey
	}
	if len(input.PortForwards) > 0 {
		attributes["port_forwards"] = input.PortForwards
	}
	if len(input.CloudInit) > 0 {
		attributes["cloud_init"] = input.CloudInit
	}
	if input.GPUType != "" && input.GPUCount > 0 {
		attributes["resources"].(map[string]any)["gpus"] = map[string]any{
			input.GPUType: map[string]any{
				"count": input.GPUCount,
			},
		}
	}

	payload := map[string]any{
		"data": map[string]any{
			"type":       "virtualmachine",
			"attributes": attributes,
		},
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
	payload := map[string]any{}
	if input.VCPUCount > 0 {
		payload["cpuCores"] = input.VCPUCount
	}
	if input.RAMGB > 0 {
		payload["ramGb"] = input.RAMGB
	}
	if input.StorageGB > 0 {
		payload["diskGb"] = input.StorageGB
	}
	if input.GPUType != "" && input.GPUCount > 0 {
		payload["gpus"] = map[string]any{
			"gpuV0Name": input.GPUType,
			"count":     input.GPUCount,
		}
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
