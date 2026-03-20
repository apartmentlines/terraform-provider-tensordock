package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeInstanceWrappedResponse(t *testing.T) {
	body := []byte(`{
		"data": {
			"id": "inst_123",
			"name": "gpu-worker-1",
			"status": "running",
			"ipAddress": "203.0.113.10",
			"rateHourly": 1.25,
			"portForwards": [
				{"internal_port": 22, "external_port": 22022}
			],
			"resources": {
				"vcpu_count": 8,
				"ram_gb": 32,
				"storage_gb": 200,
				"gpus": {
					"geforcertx4090-pcie-24gb": {
						"count": 1,
						"v0Name": "geforcertx4090-pcie-24gb"
					}
				}
			}
		}
	}`)

	instance, err := decodeInstance(body)
	if err != nil {
		t.Fatalf("decodeInstance returned error: %v", err)
	}

	if instance.ID != "inst_123" {
		t.Fatalf("unexpected ID: %q", instance.ID)
	}
	if instance.Name != "gpu-worker-1" {
		t.Fatalf("unexpected Name: %q", instance.Name)
	}
	if instance.Status != "running" {
		t.Fatalf("unexpected Status: %q", instance.Status)
	}
	if instance.IPAddress != "203.0.113.10" {
		t.Fatalf("unexpected IPAddress: %q", instance.IPAddress)
	}
	if instance.RateHourly == nil || *instance.RateHourly != 1.25 {
		t.Fatalf("unexpected RateHourly: %#v", instance.RateHourly)
	}
	if instance.VCPUCount != 8 || instance.RAMGB != 32 || instance.StorageGB != 200 {
		t.Fatalf("unexpected resources: %+v", instance)
	}
	if instance.GPUType != "geforcertx4090-pcie-24gb" || instance.GPUCount != 1 {
		t.Fatalf("unexpected GPU values: type=%q count=%d", instance.GPUType, instance.GPUCount)
	}
	if len(instance.PortForwards) != 1 || instance.PortForwards[0].ExternalPort != 22022 || instance.PortForwards[0].InternalPort != 22 {
		t.Fatalf("unexpected port forwards: %+v", instance.PortForwards)
	}
}

func TestFlattenGPUMapUsesV0NameWhenPresent(t *testing.T) {
	gpuType, gpuCount := flattenGPUMap(map[string]gpuDetails{
		"display-name": {
			Count:  2,
			V0Name: "canonical-name",
		},
	})

	if gpuType != "canonical-name" {
		t.Fatalf("unexpected gpuType: %q", gpuType)
	}
	if gpuCount != 2 {
		t.Fatalf("unexpected gpuCount: %d", gpuCount)
	}
}

func TestNormalizeStatusAndPowerState(t *testing.T) {
	cases := []struct {
		status     string
		normalized string
		powerState string
	}{
		{status: "stopped_disassociated", normalized: "stoppeddisassociated", powerState: "stopped"},
		{status: "Starting", normalized: "starting", powerState: "running"},
		{status: " running ", normalized: "running", powerState: "running"},
	}

	for _, tc := range cases {
		if got := normalizeStatus(tc.status); got != tc.normalized {
			t.Fatalf("normalizeStatus(%q) = %q, want %q", tc.status, got, tc.normalized)
		}
		if got := normalizePowerState(tc.status); got != tc.powerState {
			t.Fatalf("normalizePowerState(%q) = %q, want %q", tc.status, got, tc.powerState)
		}
	}
}

func TestDecodeInstanceRejectsUnsupportedShape(t *testing.T) {
	body, err := json.Marshal(map[string]any{"message": "no instance here"})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if _, err := decodeInstance(body); err == nil {
		t.Fatal("expected decodeInstance to fail for unsupported response shape")
	}
}

func TestClientCreateInstanceHostnodePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		attributes := payload["data"].(map[string]any)["attributes"].(map[string]any)
		if attributes["hostnode_id"] != "host-123" {
			t.Fatalf("unexpected hostnode_id: %#v", attributes["hostnode_id"])
		}
		if _, ok := attributes["location_id"]; ok {
			t.Fatalf("did not expect location_id in payload: %#v", attributes["location_id"])
		}
		if attributes["ssh_key"] != "ssh-ed25519 AAAA..." {
			t.Fatalf("unexpected ssh_key: %#v", attributes["ssh_key"])
		}
		if _, ok := attributes["useDedicatedIp"]; ok {
			t.Fatal("did not expect useDedicatedIp in payload when false")
		}

		resources := attributes["resources"].(map[string]any)
		if _, ok := resources["gpus"]; ok {
			t.Fatal("did not expect GPU payload for CPU-only hostnode deployment")
		}

		portForwards := attributes["port_forwards"].([]any)
		if len(portForwards) != 1 {
			t.Fatalf("unexpected port_forwards: %#v", portForwards)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"inst_456","name":"cpu-worker","status":"queued"}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	instance, err := client.CreateInstance(context.Background(), CreateInstanceInput{
		Name:         "cpu-worker",
		Image:        "ubuntu2404",
		HostnodeID:   "host-123",
		VCPUCount:    4,
		RAMGB:        8,
		StorageGB:    100,
		PortForwards: []PortForward{{InternalPort: 22, ExternalPort: 22022}},
		SSHPublicKey: "ssh-ed25519 AAAA...",
	})
	if err != nil {
		t.Fatalf("CreateInstance returned error: %v", err)
	}

	if instance.ID != "inst_456" || instance.Name != "cpu-worker" || instance.Status != "queued" {
		t.Fatalf("unexpected instance: %+v", instance)
	}
}

func TestClientListLocationsDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/locations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"data": {
				"locations": [
					{
						"id": "loc-1",
						"city": "Ashburn",
						"stateprovince": "VA",
						"country": "USA",
						"tier": 1,
						"gpus": [
							{
								"v0Name": "geforcertx4090-pcie-24gb",
								"displayName": "RTX 4090",
								"max_count": 2,
								"price_per_hr": 0.99,
								"resources": {"max_vcpus": 32, "max_ram_gb": 128, "max_storage_gb": 2000},
								"pricing": {"per_vcpu_hr": 0.01, "per_gb_ram_hr": 0.002, "per_gb_storage_hr": 0.0001},
								"network_features": {
									"dedicated_ip_available": true,
									"port_forwarding_available": true,
									"network_storage_available": false
								}
							}
						]
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	locations, err := client.ListLocations(context.Background())
	if err != nil {
		t.Fatalf("ListLocations returned error: %v", err)
	}

	if len(locations) != 1 || locations[0].ID != "loc-1" {
		t.Fatalf("unexpected locations: %+v", locations)
	}
	if len(locations[0].GPUs) != 1 || locations[0].GPUs[0].V0Name != "geforcertx4090-pcie-24gb" {
		t.Fatalf("unexpected location GPUs: %+v", locations[0].GPUs)
	}
}

func TestClientListHostnodesDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hostnodes" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"data": {
				"hostnodes": [
					{
						"id": "host-1",
						"location_id": "loc-1",
						"engine": "kvm",
						"uptime_percentage": 99.9,
						"available_resources": {
							"gpus": [{"v0Name": "teslav100-pcie-16gb", "availableCount": 1, "price_per_hr": 0.7}],
							"vcpu_count": 64,
							"ram_gb": 256,
							"storage_gb": 4000,
							"available_ports": [22022, 22080],
							"has_public_ip_available": true
						},
						"pricing": {"per_vcpu_hr": 0.01, "per_gb_ram_hr": 0.002, "per_gb_storage_hr": 0.0001},
						"location": {
							"uuid": "loc-1",
							"city": "Ashburn",
							"stateprovince": "VA",
							"country": "USA",
							"has_network_storage": true,
							"network_speed_gbps": 2.5,
							"network_speed_upload_gbps": 10.5,
							"organization": "org",
							"organizationName": "Org",
							"tier": 1
						}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	hostnodes, err := client.ListHostnodes(context.Background())
	if err != nil {
		t.Fatalf("ListHostnodes returned error: %v", err)
	}

	if len(hostnodes) != 1 || hostnodes[0].ID != "host-1" {
		t.Fatalf("unexpected hostnodes: %+v", hostnodes)
	}
	if got := hostnodes[0].AvailableResources.GPUs[0].V0Name; got != "teslav100-pcie-16gb" {
		t.Fatalf("unexpected hostnode GPU name: %q", got)
	}
	if got := hostnodes[0].Location.NetworkSpeedGbps; got != 2.5 {
		t.Fatalf("unexpected network speed: %v", got)
	}
}

func TestClientSecretMethodsDecodeResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/secrets":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode secret create request: %v", err)
			}

			attributes := payload["data"].(map[string]any)["attributes"].(map[string]any)
			if attributes["name"] != "deploy-key" || attributes["type"] != "ssh" || attributes["value"] != "secret-value" {
				t.Fatalf("unexpected secret create payload: %#v", attributes)
			}

			_, _ = w.Write([]byte(`{"data":{"id":"secret-1","name":"deploy-key","type":"ssh"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/secrets/secret-1":
			_, _ = w.Write([]byte(`{"data":{"id":"secret-1","name":"deploy-key","type":"ssh","value":"redacted-or-present"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token", "test")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	created, err := client.CreateSecret(context.Background(), "deploy-key", "ssh", "secret-value")
	if err != nil {
		t.Fatalf("CreateSecret returned error: %v", err)
	}
	if created.ID != "secret-1" || created.Name != "deploy-key" {
		t.Fatalf("unexpected created secret: %+v", created)
	}

	secret, err := client.GetSecret(context.Background(), "secret-1")
	if err != nil {
		t.Fatalf("GetSecret returned error: %v", err)
	}
	if secret.Value == "" || !strings.Contains(secret.Value, "redacted") && !strings.Contains(secret.Value, "present") {
		t.Fatalf("unexpected secret value: %+v", secret)
	}
}
