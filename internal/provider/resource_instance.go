package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &InstanceResource{}
	_ resource.ResourceWithImportState = &InstanceResource{}
)

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

type InstanceResource struct {
	client *Client
}

type InstanceResourceModel struct {
	ID             types.String  `tfsdk:"id"`
	Name           types.String  `tfsdk:"name"`
	Image          types.String  `tfsdk:"image"`
	LocationID     types.String  `tfsdk:"location_id"`
	VCPUCount      types.Int64   `tfsdk:"vcpu_count"`
	RAMGB          types.Int64   `tfsdk:"ram_gb"`
	StorageGB      types.Int64   `tfsdk:"storage_gb"`
	GPUType        types.String  `tfsdk:"gpu_type"`
	GPUCount       types.Int64   `tfsdk:"gpu_count"`
	UseDedicatedIP types.Bool    `tfsdk:"use_dedicated_ip"`
	SSHPublicKey   types.String  `tfsdk:"ssh_public_key"`
	CloudInitJSON  types.String  `tfsdk:"cloud_init_json"`
	PowerState     types.String  `tfsdk:"power_state"`
	Status         types.String  `tfsdk:"status"`
	IPAddress      types.String  `tfsdk:"ip_address"`
	RateHourly     types.Float64 `tfsdk:"rate_hourly"`
	PortForwards   types.List    `tfsdk:"port_forwards"`
}

func (r *InstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a TensorDock virtual machine instance using the public v2 API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "TensorDock instance identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Instance name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "TensorDock image identifier, for example `ubuntu2404`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location_id": schema.StringAttribute{
				MarkdownDescription: "TensorDock location UUID used for location-based deployment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vcpu_count": schema.Int64Attribute{
				MarkdownDescription: "Requested vCPU count.",
				Required:            true,
			},
			"ram_gb": schema.Int64Attribute{
				MarkdownDescription: "Requested memory in GiB.",
				Required:            true,
			},
			"storage_gb": schema.Int64Attribute{
				MarkdownDescription: "Requested storage in GiB. TensorDock documents a minimum of 100GB and only supports increasing storage through the modify endpoint.",
				Required:            true,
			},
			"gpu_type": schema.StringAttribute{
				MarkdownDescription: "TensorDock GPU model v0Name, for example `geforcertx4090-pcie-24gb`.",
				Required:            true,
			},
			"gpu_count": schema.Int64Attribute{
				MarkdownDescription: "Number of GPUs of `gpu_type`.",
				Required:            true,
			},
			"use_dedicated_ip": schema.BoolAttribute{
				MarkdownDescription: "Request a dedicated IP during instance creation. TensorDock documents this as a create-time field.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
				},
			},
			"ssh_public_key": schema.StringAttribute{
				MarkdownDescription: "SSH public key injected during instance creation. TensorDock documents this as a create-time field.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_init_json": schema.StringAttribute{
				MarkdownDescription: "Optional JSON representation of TensorDock's documented `cloud_init` object. This is treated as a create-time field.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"power_state": schema.StringAttribute{
				MarkdownDescription: "Desired power state. Supported values are `running` and `stopped`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Raw instance status returned by TensorDock.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "Instance IP address returned by TensorDock.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rate_hourly": schema.Float64Attribute{
				MarkdownDescription: "Hourly rate returned by TensorDock.",
				Computed:            true,
			},
			"port_forwards": schema.ListNestedAttribute{
				MarkdownDescription: "Port forwards returned by TensorDock for the instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"internal_port": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Internal port on the virtual machine.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"external_port": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Externally exposed port.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *InstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *provider.Client, got %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cloudInit, desiredPowerState, useDedicatedIP := normalizeInstancePlan(&plan)
	resp.Diagnostics.Append(validateInstancePlan(plan, desiredPowerState, cloudInit)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateInstance(ctx, CreateInstanceInput{
		Name:           plan.Name.ValueString(),
		Image:          plan.Image.ValueString(),
		LocationID:     plan.LocationID.ValueString(),
		VCPUCount:      plan.VCPUCount.ValueInt64(),
		RAMGB:          plan.RAMGB.ValueInt64(),
		StorageGB:      plan.StorageGB.ValueInt64(),
		GPUType:        plan.GPUType.ValueString(),
		GPUCount:       plan.GPUCount.ValueInt64(),
		UseDedicatedIP: useDedicatedIP,
		SSHPublicKey:   plan.SSHPublicKey.ValueString(),
		CloudInit:      cloudInit,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create TensorDock instance", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)

	remote, err := r.client.WaitForStatus(ctx, created.ID, "running")
	if err != nil {
		resp.Diagnostics.AddError("Unable to observe created TensorDock instance", err.Error())
		return
	}

	if desiredPowerState == "stopped" {
		if err := r.client.StopInstance(ctx, created.ID); err != nil {
			resp.Diagnostics.AddError("Unable to stop TensorDock instance after creation", err.Error())
			return
		}

		remote, err = r.client.WaitForStatus(ctx, created.ID, "stopped", "stoppeddisassociated")
		if err != nil {
			resp.Diagnostics.AddError("Unable to confirm stopped TensorDock instance", err.Error())
			return
		}
	}

	plan.UseDedicatedIP = types.BoolValue(useDedicatedIP)
	syncModelFromRemote(&plan, remote)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, err := r.client.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Unable to read TensorDock instance", err.Error())
		return
	}

	syncModelFromRemote(&state, remote)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceResourceModel
	var state InstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cloudInit, desiredPowerState, useDedicatedIP := normalizeInstancePlan(&plan)
	plan.UseDedicatedIP = types.BoolValue(useDedicatedIP)

	resp.Diagnostics.Append(validateInstancePlan(plan, desiredPowerState, cloudInit)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.StorageGB.ValueInt64() > 0 && plan.StorageGB.ValueInt64() < state.StorageGB.ValueInt64() {
		resp.Diagnostics.AddError(
			"TensorDock does not support shrinking instance storage in place",
			"The public TensorDock modify endpoint only allows storage to be increased. Recreate the instance to apply a smaller `storage_gb` value.",
		)
		return
	}

	remote, err := r.client.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Unable to read TensorDock instance before update", err.Error())
		return
	}

	if hasHardwareChange(state, plan) {
		remote, err = r.ensureStopped(ctx, state.ID.ValueString(), remote)
		if err != nil {
			resp.Diagnostics.AddError("Unable to prepare TensorDock instance for resize", err.Error())
			return
		}

		err = r.client.ModifyInstance(ctx, state.ID.ValueString(), ModifyInstanceInput{
			VCPUCount: plan.VCPUCount.ValueInt64(),
			RAMGB:     plan.RAMGB.ValueInt64(),
			StorageGB: plan.StorageGB.ValueInt64(),
			GPUType:   plan.GPUType.ValueString(),
			GPUCount:  plan.GPUCount.ValueInt64(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Unable to modify TensorDock instance", err.Error())
			return
		}

		remote, err = r.client.WaitForStatus(ctx, state.ID.ValueString(), "stopped", "stoppeddisassociated")
		if err != nil {
			resp.Diagnostics.AddError("Unable to confirm modified TensorDock instance state", err.Error())
			return
		}
	}

	remote, err = r.reconcilePowerState(ctx, state.ID.ValueString(), remote, desiredPowerState)
	if err != nil {
		resp.Diagnostics.AddError("Unable to reconcile TensorDock power state", err.Error())
		return
	}

	syncModelFromRemote(&plan, remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteInstance(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete TensorDock instance", err.Error())
		return
	}

	if err := r.client.WaitForDeletion(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to confirm TensorDock instance deletion", err.Error())
		return
	}
}

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *InstanceResource) ensureStopped(ctx context.Context, id string, current Instance) (Instance, error) {
	if isStoppedStatus(current.Status) {
		return current, nil
	}

	if err := r.client.StopInstance(ctx, id); err != nil {
		return Instance{}, err
	}

	return r.client.WaitForStatus(ctx, id, "stopped", "stoppeddisassociated")
}

func (r *InstanceResource) reconcilePowerState(ctx context.Context, id string, current Instance, desired string) (Instance, error) {
	switch desired {
	case "running":
		if normalizePowerState(current.Status) == "running" {
			return current, nil
		}
		if err := r.client.StartInstance(ctx, id); err != nil {
			return Instance{}, err
		}
		return r.client.WaitForStatus(ctx, id, "running")
	case "stopped":
		if normalizePowerState(current.Status) == "stopped" {
			return current, nil
		}
		if err := r.client.StopInstance(ctx, id); err != nil {
			return Instance{}, err
		}
		return r.client.WaitForStatus(ctx, id, "stopped", "stoppeddisassociated")
	default:
		return current, nil
	}
}

func validateInstancePlan(plan InstanceResourceModel, desiredPowerState string, cloudInit map[string]any) diag.Diagnostics {
	var diags diag.Diagnostics

	if plan.Name.IsNull() || plan.Name.IsUnknown() || strings.TrimSpace(plan.Name.ValueString()) == "" {
		diags.AddError("Missing instance name", "`name` must be a non-empty string.")
	}
	if plan.Image.IsNull() || plan.Image.IsUnknown() || strings.TrimSpace(plan.Image.ValueString()) == "" {
		diags.AddError("Missing image", "`image` must be a non-empty string.")
	}
	if plan.LocationID.IsNull() || plan.LocationID.IsUnknown() || strings.TrimSpace(plan.LocationID.ValueString()) == "" {
		diags.AddError("Missing location ID", "`location_id` must be a non-empty string.")
	}
	if plan.SSHPublicKey.IsNull() || plan.SSHPublicKey.IsUnknown() || strings.TrimSpace(plan.SSHPublicKey.ValueString()) == "" {
		diags.AddError("Missing SSH public key", "`ssh_public_key` must be supplied because TensorDock documents SSH keys as required during instance creation.")
	}
	if plan.VCPUCount.IsNull() || plan.VCPUCount.IsUnknown() || plan.VCPUCount.ValueInt64() <= 0 {
		diags.AddError("Invalid vCPU count", "`vcpu_count` must be greater than zero.")
	}
	if plan.RAMGB.IsNull() || plan.RAMGB.IsUnknown() || plan.RAMGB.ValueInt64() <= 0 {
		diags.AddError("Invalid RAM size", "`ram_gb` must be greater than zero.")
	}
	if plan.StorageGB.IsNull() || plan.StorageGB.IsUnknown() || plan.StorageGB.ValueInt64() < 100 {
		diags.AddError("Invalid storage size", "TensorDock documents a minimum of 100GB for instance creation.")
	}
	if plan.GPUType.IsNull() || plan.GPUType.IsUnknown() || strings.TrimSpace(plan.GPUType.ValueString()) == "" {
		diags.AddError("Missing GPU type", "`gpu_type` must be supplied for location-based deployments.")
	}
	if plan.GPUCount.IsNull() || plan.GPUCount.IsUnknown() || plan.GPUCount.ValueInt64() < 1 {
		diags.AddError("Invalid GPU count", "TensorDock documents that at least one GPU is required for location-based deployment.")
	}
	if desiredPowerState != "running" && desiredPowerState != "stopped" {
		diags.AddError("Invalid power_state", "`power_state` must be either `running` or `stopped`.")
	}
	if plan.CloudInitJSON.IsUnknown() {
		diags.AddError("Invalid cloud_init_json", "`cloud_init_json` must be known during planning if supplied.")
	}
	if !plan.CloudInitJSON.IsNull() && strings.TrimSpace(plan.CloudInitJSON.ValueString()) != "" && cloudInit == nil {
		diags.AddError("Invalid cloud_init_json", "`cloud_init_json` must decode to a JSON object.")
	}

	return diags
}

func normalizeInstancePlan(plan *InstanceResourceModel) (map[string]any, string, bool) {
	desiredPowerState := "running"
	if !plan.PowerState.IsNull() && !plan.PowerState.IsUnknown() {
		desiredPowerState = strings.TrimSpace(strings.ToLower(plan.PowerState.ValueString()))
	}
	plan.PowerState = types.StringValue(desiredPowerState)

	useDedicatedIP := false
	if !plan.UseDedicatedIP.IsNull() && !plan.UseDedicatedIP.IsUnknown() {
		useDedicatedIP = plan.UseDedicatedIP.ValueBool()
	}
	plan.UseDedicatedIP = types.BoolValue(useDedicatedIP)

	cloudInit := map[string]any(nil)
	if !plan.CloudInitJSON.IsNull() && !plan.CloudInitJSON.IsUnknown() {
		trimmed := strings.TrimSpace(plan.CloudInitJSON.ValueString())
		if trimmed != "" {
			var candidate map[string]any
			if err := json.Unmarshal([]byte(trimmed), &candidate); err == nil {
				cloudInit = candidate
			}
		}
	}

	return cloudInit, desiredPowerState, useDedicatedIP
}

func hasHardwareChange(state, plan InstanceResourceModel) bool {
	return state.VCPUCount.ValueInt64() != plan.VCPUCount.ValueInt64() ||
		state.RAMGB.ValueInt64() != plan.RAMGB.ValueInt64() ||
		state.StorageGB.ValueInt64() != plan.StorageGB.ValueInt64() ||
		state.GPUCount.ValueInt64() != plan.GPUCount.ValueInt64() ||
		state.GPUType.ValueString() != plan.GPUType.ValueString()
}

func syncModelFromRemote(model *InstanceResourceModel, remote Instance) {
	if remote.ID != "" {
		model.ID = types.StringValue(remote.ID)
	}
	if remote.Name != "" {
		model.Name = types.StringValue(remote.Name)
	}
	if remote.Status != "" {
		model.Status = types.StringValue(strings.TrimSpace(remote.Status))
		normalizedPowerState := normalizePowerState(remote.Status)
		if normalizedPowerState != "" {
			model.PowerState = types.StringValue(normalizedPowerState)
		}
	}
	if remote.IPAddress != "" {
		model.IPAddress = types.StringValue(remote.IPAddress)
	} else {
		model.IPAddress = types.StringNull()
	}
	if remote.RateHourly != nil {
		model.RateHourly = types.Float64Value(*remote.RateHourly)
	} else {
		model.RateHourly = types.Float64Null()
	}
	if remote.VCPUCount > 0 {
		model.VCPUCount = types.Int64Value(remote.VCPUCount)
	}
	if remote.RAMGB > 0 {
		model.RAMGB = types.Int64Value(remote.RAMGB)
	}
	if remote.StorageGB > 0 {
		model.StorageGB = types.Int64Value(remote.StorageGB)
	}
	if remote.GPUType != "" {
		model.GPUType = types.StringValue(remote.GPUType)
	}
	if remote.GPUCount > 0 {
		model.GPUCount = types.Int64Value(remote.GPUCount)
	}

	model.PortForwards = buildPortForwardList(remote.PortForwards)
}

func buildPortForwardList(portForwards []PortForward) types.List {
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"internal_port": types.Int64Type,
		"external_port": types.Int64Type,
	}}

	if len(portForwards) == 0 {
		empty, diags := types.ListValue(objectType, []attr.Value{})
		if diags.HasError() {
			return types.ListNull(objectType)
		}
		return empty
	}

	sorted := append([]PortForward(nil), portForwards...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ExternalPort == sorted[j].ExternalPort {
			return sorted[i].InternalPort < sorted[j].InternalPort
		}
		return sorted[i].ExternalPort < sorted[j].ExternalPort
	})

	values := make([]attr.Value, 0, len(sorted))
	for _, pf := range sorted {
		objectValue, diags := types.ObjectValue(objectType.AttrTypes, map[string]attr.Value{
			"internal_port": types.Int64Value(pf.InternalPort),
			"external_port": types.Int64Value(pf.ExternalPort),
		})
		if diags.HasError() {
			return types.ListNull(objectType)
		}
		values = append(values, objectValue)
	}

	listValue, diags := types.ListValue(objectType, values)
	if diags.HasError() {
		return types.ListNull(objectType)
	}

	return listValue
}

func isStoppedStatus(status string) bool {
	normalized := normalizeStatus(status)
	return normalized == "stopped" || normalized == "stoppeddisassociated"
}
