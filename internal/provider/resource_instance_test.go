package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateInstancePlanRequiresExactlyOnePlacementTarget(t *testing.T) {
	base := validLocationPlan()
	base.LocationID = types.StringNull()

	diags := validateInstancePlan(base, "running", nil, nil)
	if !diags.HasError() {
		t.Fatal("expected diagnostics when neither location_id nor hostnode_id is set")
	}

	base = validLocationPlan()
	base.HostnodeID = types.StringValue("host-1")

	diags = validateInstancePlan(base, "running", nil, nil)
	if !diags.HasError() {
		t.Fatal("expected diagnostics when both location_id and hostnode_id are set")
	}
}

func TestValidateInstancePlanRequiresGPUForLocationPlacement(t *testing.T) {
	plan := validLocationPlan()
	plan.GPUType = types.StringValue("")
	plan.GPUCount = types.Int64Value(0)

	diags := validateInstancePlan(plan, "running", nil, nil)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for location deployment without GPUs")
	}
}

func TestValidateInstancePlanAllowsCPUOnlyHostnodePlacement(t *testing.T) {
	plan := validLocationPlan()
	plan.LocationID = types.StringNull()
	plan.HostnodeID = types.StringValue("host-1")
	plan.GPUType = types.StringValue("")
	plan.GPUCount = types.Int64Value(0)

	diags := validateInstancePlan(plan, "running", nil, nil)
	if diags.HasError() {
		t.Fatalf("expected no diagnostics, got: %+v", diags)
	}
}

func TestValidateInstancePlanWindowsImageDoesNotRequireSSHKey(t *testing.T) {
	plan := validLocationPlan()
	plan.Image = types.StringValue("windows10")
	plan.SSHPublicKey = types.StringNull()

	diags := validateInstancePlan(plan, "running", nil, nil)
	if diags.HasError() {
		t.Fatalf("expected no diagnostics, got: %+v", diags)
	}
}

func TestValidateInstanceUpdateRejectsUnsupportedModifyShapes(t *testing.T) {
	state := validLocationPlan()
	plan := validLocationPlan()

	plan.VCPUCount = types.Int64Value(5)
	diags := validateInstanceUpdate(state, plan)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for odd CPU count during modify")
	}

	plan = validLocationPlan()
	plan.RAMGB = types.Int64Value(12)
	diags = validateInstanceUpdate(state, plan)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for invalid RAM size during modify")
	}

	plan = validLocationPlan()
	plan.StorageGB = types.Int64Value(50)
	diags = validateInstanceUpdate(state, plan)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for storage shrink during modify")
	}

	plan = validLocationPlan()
	plan.GPUType = types.StringValue("")
	plan.GPUCount = types.Int64Value(0)
	diags = validateInstanceUpdate(state, plan)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for removing GPUs in place")
	}
}

func validLocationPlan() InstanceResourceModel {
	return InstanceResourceModel{
		Name:           types.StringValue("worker"),
		Image:          types.StringValue("ubuntu2404"),
		LocationID:     types.StringValue("loc-1"),
		HostnodeID:     types.StringNull(),
		VCPUCount:      types.Int64Value(8),
		RAMGB:          types.Int64Value(32),
		StorageGB:      types.Int64Value(200),
		GPUType:        types.StringValue("geforcertx4090-pcie-24gb"),
		GPUCount:       types.Int64Value(1),
		UseDedicatedIP: types.BoolValue(false),
		SSHPublicKey:   types.StringValue("ssh-ed25519 AAAA..."),
	}
}
