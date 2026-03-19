package provider

import (
	"encoding/json"
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
