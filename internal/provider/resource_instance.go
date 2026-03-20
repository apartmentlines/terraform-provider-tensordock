package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

type PortForwardModel struct {
	InternalPort types.Int64 `tfsdk:"internal_port"`
	ExternalPort types.Int64 `tfsdk:"external_port"`
}

type InstanceResourceModel struct {
	ID             types.String  `tfsdk:"id"`
	Name           types.String  `tfsdk:"name"`
	Image          types.String  `tfsdk:"image"`
	LocationID     types.String  `tfsdk:"location_id"`
	HostnodeID     types.String  `tfsdk:"hostnode_id"`
	VCPUCount      types.Int64   `tfsdk:"vcpu_count"`
	RAMGB          types.Int64   `tfsdk:"ram_gb"`
	StorageGB      types.Int64   `tfsdk:"storage_gb"`
	GPUType        types.String  `tfsdk:"gpu_type"`
	GPUCount       types.Int64   `tfsdk:"gpu_count"`
	UseDedicatedIP types.Bool    `tfsdk:"use_dedicated_ip"`
	PortForwards   types.List    `tfsdk:"port_forwards"`
	SSHPublicKey   types.String  `tfsdk:"ssh_public_key"`
	CloudInitJSON  types.String  `tfsdk:"cloud_init_json"`
	PowerState     types.String  `tfsdk:"power_state"`
	Status         types.String  `tfsdk:"status"`
	IPAddress      types.String  `tfsdk:"ip_address"`
	RateHourly     types.Float64 `tfsdk:"rate_hourly"`
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
				MarkdownDescription: "TensorDock location UUID used for location-based deployment. Exactly one of `location_id` or `hostnode_id` must be set.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hostnode_id": schema.StringAttribute{
				MarkdownDescription: "TensorDock hostnode UUID used for direct hostnode deployment. Exactly one of `location_id` or `hostnode_id` must be set.",
				Optional:            true,
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
				MarkdownDescription: "TensorDock GPU model v0Name, for example `geforcertx4090-pcie-24gb`. Required for location-based deployments.",
				Optional:            true,
			},
			"gpu_count": schema.Int64Attribute{
				MarkdownDescription: "Number of GPUs of `gpu_type`. Required for location-based deployments.",
				Optional:            true,
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
			"port_forwards": schema.ListNestedAttribute{
				MarkdownDescription: "Optional create-time port forward mappings. TensorDock also returns current port forwards on instance reads.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"internal_port": schema.Int64Attribute{
							MarkdownDescription: "Internal port on the virtual machine.",
							Required:            true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"external_port": schema.Int64Attribute{
							MarkdownDescription: "Externally exposed port.",
							Required:            true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"ssh_public_key": schema.StringAttribute{
				MarkdownDescription: "SSH public key injected during instance creation. This attribute is write-only and is not stored in Terraform state. Required for non-Windows images.",
				Optional:            true,
				WriteOnly:           true,
				Sensitive:           true,
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

	cloudInit, desiredPowerState, useDedicatedIP, portForwards := normalizeInstancePlan(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateInstancePlan(plan, desiredPowerState, cloudInit, portForwards)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateInstance(ctx, CreateInstanceInput{
		Name:           strings.TrimSpace(plan.Name.ValueString()),
		Image:          strings.TrimSpace(plan.Image.ValueString()),
		LocationID:     strings.TrimSpace(plan.LocationID.ValueString()),
		HostnodeID:     strings.TrimSpace(plan.HostnodeID.ValueString()),
		VCPUCount:      plan.VCPUCount.ValueInt64(),
		RAMGB:          plan.RAMGB.ValueInt64(),
		StorageGB:      plan.StorageGB.ValueInt64(),
		GPUType:        strings.TrimSpace(plan.GPUType.ValueString()),
		GPUCount:       plan.GPUCount.ValueInt64(),
		UseDedicatedIP: useDedicatedIP,
		PortForwards:   portForwards,
		SSHPublicKey:   strings.TrimSpace(plan.SSHPublicKey.ValueString()),
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
	plan.PortForwards = buildPortForwardList(portForwards)
	plan.SSHPublicKey = types.StringNull()
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
	state.SSHPublicKey = types.StringNull()

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

	cloudInit, desiredPowerState, useDedicatedIP, portForwards := normalizeInstancePlan(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateInstancePlan(plan, desiredPowerState, cloudInit, portForwards)...)
	resp.Diagnostics.Append(validateInstanceUpdate(state, plan)...)
	if resp.Diagnostics.HasError() {
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
			GPUType:   strings.TrimSpace(plan.GPUType.ValueString()),
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

	plan.UseDedicatedIP = types.BoolValue(useDedicatedIP)
	plan.PortForwards = buildPortForwardList(portForwards)
	plan.SSHPublicKey = types.StringNull()
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

func validateInstancePlan(plan InstanceResourceModel, desiredPowerState string, cloudInit map[string]any, portForwards []PortForward) diag.Diagnostics {
	var diags diag.Diagnostics

	name := strings.TrimSpace(plan.Name.ValueString())
	image := strings.TrimSpace(plan.Image.ValueString())
	locationID := strings.TrimSpace(plan.LocationID.ValueString())
	hostnodeID := strings.TrimSpace(plan.HostnodeID.ValueString())
	sshPublicKey := strings.TrimSpace(plan.SSHPublicKey.ValueString())
	gpuType := strings.TrimSpace(plan.GPUType.ValueString())
	gpuCount := plan.GPUCount.ValueInt64()

	if plan.Name.IsNull() || plan.Name.IsUnknown() || name == "" {
		diags.AddError("Missing instance name", "`name` must be a non-empty string.")
	}
	if plan.Image.IsNull() || plan.Image.IsUnknown() || image == "" {
		diags.AddError("Missing image", "`image` must be a non-empty string.")
	}
	if locationID == "" && hostnodeID == "" {
		diags.AddError("Missing placement target", "Exactly one of `location_id` or `hostnode_id` must be supplied.")
	}
	if locationID != "" && hostnodeID != "" {
		diags.AddError("Conflicting placement target", "`location_id` and `hostnode_id` are mutually exclusive.")
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
	if gpuCount < 0 {
		diags.AddError("Invalid GPU count", "`gpu_count` cannot be negative.")
	}
	if gpuType == "" && gpuCount > 0 {
		diags.AddError("Missing GPU type", "`gpu_type` must be supplied when `gpu_count` is greater than zero.")
	}
	if gpuType != "" && gpuCount < 1 {
		diags.AddError("Invalid GPU count", "`gpu_count` must be at least 1 when `gpu_type` is supplied.")
	}
	if locationID != "" {
		if gpuType == "" {
			diags.AddError("Missing GPU type", "`gpu_type` must be supplied for location-based deployment.")
		}
		if gpuCount < 1 {
			diags.AddError("Invalid GPU count", "TensorDock location-based deployment requires at least one GPU.")
		}
	}
	if requiresSSHKey(image) && sshPublicKey == "" {
		diags.AddError("Missing SSH public key", "`ssh_public_key` must be supplied for non-Windows images.")
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

	for _, portForward := range portForwards {
		if !validPort(portForward.InternalPort) || !validPort(portForward.ExternalPort) {
			diags.AddError(
				"Invalid port_forwards entry",
				"`port_forwards` values must use ports between 1 and 65535.",
			)
			break
		}
	}

	return diags
}

func validateInstanceUpdate(state, plan InstanceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if state.StorageGB.ValueInt64() > 0 && plan.StorageGB.ValueInt64() < state.StorageGB.ValueInt64() {
		diags.AddError(
			"TensorDock does not support shrinking instance storage in place",
			"The public TensorDock modify endpoint only allows storage to be increased. Recreate the instance to apply a smaller `storage_gb` value.",
		)
	}

	if state.VCPUCount.ValueInt64() != plan.VCPUCount.ValueInt64() && plan.VCPUCount.ValueInt64()%2 != 0 {
		diags.AddError("Invalid vCPU count for modification", "The documented TensorDock modify endpoint requires CPU changes to use a multiple of 2 cores.")
	}

	if state.RAMGB.ValueInt64() != plan.RAMGB.ValueInt64() && !validModifyRAM(plan.RAMGB.ValueInt64()) {
		diags.AddError("Invalid RAM size for modification", "The documented TensorDock modify endpoint only accepts specific RAM sizes.")
	}

	if state.GPUCount.ValueInt64() > 0 && plan.GPUCount.ValueInt64() == 0 {
		diags.AddError(
			"Removing GPUs is not supported in place",
			"The verified TensorDock modify payload only supports GPU changes when both `gpu_type` and `gpu_count` are supplied. Recreate the instance to remove GPUs.",
		)
	}

	return diags
}

func normalizeInstancePlan(ctx context.Context, plan *InstanceResourceModel, diags *diag.Diagnostics) (map[string]any, string, bool, []PortForward) {
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

	if plan.GPUType.IsNull() || plan.GPUType.IsUnknown() {
		plan.GPUType = types.StringValue("")
	}
	if plan.GPUCount.IsNull() || plan.GPUCount.IsUnknown() {
		plan.GPUCount = types.Int64Value(0)
	}

	portForwards := []PortForward{}
	if !plan.PortForwards.IsNull() && !plan.PortForwards.IsUnknown() {
		var planPortForwards []PortForwardModel
		diags.Append(plan.PortForwards.ElementsAs(ctx, &planPortForwards, false)...)
		if diags.HasError() {
			return nil, desiredPowerState, useDedicatedIP, nil
		}

		portForwards = make([]PortForward, 0, len(planPortForwards))
		for _, portForward := range planPortForwards {
			portForwards = append(portForwards, PortForward{
				InternalPort: portForward.InternalPort.ValueInt64(),
				ExternalPort: portForward.ExternalPort.ValueInt64(),
			})
		}
	}
	plan.PortForwards = buildPortForwardList(portForwards)

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

	return cloudInit, desiredPowerState, useDedicatedIP, portForwards
}

func hasHardwareChange(state, plan InstanceResourceModel) bool {
	return state.VCPUCount.ValueInt64() != plan.VCPUCount.ValueInt64() ||
		state.RAMGB.ValueInt64() != plan.RAMGB.ValueInt64() ||
		state.StorageGB.ValueInt64() != plan.StorageGB.ValueInt64() ||
		state.GPUCount.ValueInt64() != plan.GPUCount.ValueInt64() ||
		strings.TrimSpace(state.GPUType.ValueString()) != strings.TrimSpace(plan.GPUType.ValueString())
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
	} else {
		model.GPUType = types.StringValue("")
	}
	if remote.GPUCount > 0 {
		model.GPUCount = types.Int64Value(remote.GPUCount)
	} else {
		model.GPUCount = types.Int64Value(0)
	}

	model.PortForwards = buildPortForwardList(remote.PortForwards)
}

func isStoppedStatus(status string) bool {
	normalized := normalizeStatus(status)
	return normalized == "stopped" || normalized == "stoppeddisassociated"
}

func validModifyRAM(value int64) bool {
	allowed := map[int64]bool{
		2: true, 4: true, 6: true, 8: true, 10: true, 16: true, 32: true, 48: true,
		64: true, 80: true, 96: true, 112: true, 128: true, 144: true, 160: true,
		176: true, 192: true, 208: true, 224: true, 240: true, 256: true, 512: true,
	}
	return allowed[value]
}

func validPort(value int64) bool {
	return value >= 1 && value <= 65535
}

func requiresSSHKey(image string) bool {
	return !strings.HasPrefix(strings.ToLower(strings.TrimSpace(image)), "windows")
}
