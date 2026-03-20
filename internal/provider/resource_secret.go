package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &SecretResource{}
	_ resource.ResourceWithImportState = &SecretResource{}
	_ resource.ResourceWithModifyPlan  = &SecretResource{}
)

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

type SecretResource struct {
	client *Client
}

type SecretResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	ValueWO        types.String `tfsdk:"value_wo"`
	ValueWOVersion types.Int64  `tfsdk:"value_wo_version"`
}

func (r *SecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage TensorDock secrets through the public v2 API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "TensorDock secret identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Secret name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Secret type.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value_wo": schema.StringAttribute{
				MarkdownDescription: "Sensitive secret value used during creation or replacement. This attribute is write-only and is not stored in Terraform state.",
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"value_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Version token that Terraform persists to detect secret rotation. Increment this value whenever `value_wo` changes.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *SecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecretResourceModel
	var valueWO types.String

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("value_wo"), &valueWO)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := strings.TrimSpace(plan.Name.ValueString())
	secretType := strings.TrimSpace(plan.Type.ValueString())
	value := strings.TrimSpace(valueWO.ValueString())

	resp.Diagnostics.Append(validateSecretPlan(plan, value)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSecret(ctx, name, secretType, value)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create TensorDock secret", err.Error())
		return
	}

	state := SecretResourceModel{
		ID:             types.StringValue(created.ID),
		Name:           types.StringValue(created.Name),
		Type:           types.StringValue(created.Type),
		ValueWO:        types.StringNull(),
		ValueWOVersion: plan.ValueWOVersion,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, err := r.client.GetSecret(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Unable to read TensorDock secret", err.Error())
		return
	}

	state.ID = types.StringValue(remote.ID)
	state.Name = types.StringValue(remote.Name)
	state.Type = types.StringValue(remote.Type)
	state.ValueWO = types.StringNull()

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ValueWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSecret(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete TensorDock secret", err.Error())
		return
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SecretResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan SecretResourceModel
	var valueWO types.String

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("value_wo"), &valueWO)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if req.State.Raw.IsNull() {
		plan = normalizeSecretCreatePlan(plan, valueWO)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
		return
	}

	var state SecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	resp.RequiresReplace = requiresReplaceSecretValueRotation(plan, state, valueWO)
}

func validateSecretPlan(plan SecretResourceModel, value string) diag.Diagnostics {
	var diags diag.Diagnostics

	name := strings.TrimSpace(plan.Name.ValueString())
	secretType := strings.TrimSpace(plan.Type.ValueString())

	if name == "" {
		diags.AddError("Missing secret name", "`name` must be supplied when creating a TensorDock secret.")
	}
	if secretType == "" {
		diags.AddError("Missing secret type", "`type` must be supplied when creating a TensorDock secret.")
	}
	if value == "" {
		diags.AddError("Missing secret value", "`value_wo` must be supplied when creating a TensorDock secret.")
	}

	return diags
}

func normalizeSecretCreatePlan(plan SecretResourceModel, valueWO types.String) SecretResourceModel {
	hasValueInput := !valueWO.IsNull() && !valueWO.IsUnknown() && strings.TrimSpace(valueWO.ValueString()) != ""

	if hasValueInput && (plan.ValueWOVersion.IsNull() || plan.ValueWOVersion.IsUnknown()) {
		plan.ValueWOVersion = types.Int64Value(1)
	}

	return plan
}

func requiresReplaceSecretValueRotation(plan, state SecretResourceModel, valueWO types.String) path.Paths {
	hasExistingState := !state.ID.IsNull() && !state.ID.IsUnknown() && strings.TrimSpace(state.ID.ValueString()) != ""
	hasValueInput := !valueWO.IsNull() && !valueWO.IsUnknown() && strings.TrimSpace(valueWO.ValueString()) != ""
	versionChanged := !plan.ValueWOVersion.Equal(state.ValueWOVersion)

	if hasExistingState && hasValueInput && versionChanged {
		return path.Paths{path.Root("value_wo_version")}
	}

	return nil
}
