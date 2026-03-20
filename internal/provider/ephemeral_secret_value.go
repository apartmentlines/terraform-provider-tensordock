package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ephemeral.EphemeralResource              = &SecretValueEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &SecretValueEphemeralResource{}
)

func NewSecretValueEphemeralResource() ephemeral.EphemeralResource {
	return &SecretValueEphemeralResource{}
}

type SecretValueEphemeralResource struct {
	client *Client
}

type SecretValueEphemeralModel struct {
	SecretID types.String `tfsdk:"secret_id"`
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Value    types.String `tfsdk:"value"`
}

func (r *SecretValueEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_value"
}

func (r *SecretValueEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = ephemeralschema.Schema{
		MarkdownDescription: "Fetch a TensorDock secret value for use during a Terraform run without persisting it to state.",
		Attributes: map[string]ephemeralschema.Attribute{
			"secret_id": ephemeralschema.StringAttribute{
				MarkdownDescription: "TensorDock secret ID to fetch.",
				Required:            true,
			},
			"id": ephemeralschema.StringAttribute{
				MarkdownDescription: "TensorDock secret ID.",
				Computed:            true,
			},
			"name": ephemeralschema.StringAttribute{
				MarkdownDescription: "TensorDock secret name.",
				Computed:            true,
			},
			"type": ephemeralschema.StringAttribute{
				MarkdownDescription: "TensorDock secret type.",
				Computed:            true,
			},
			"value": ephemeralschema.StringAttribute{
				MarkdownDescription: "Sensitive secret value fetched live from the TensorDock API for this run only.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *SecretValueEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *SecretValueEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var config SecretValueEphemeralModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretID := strings.TrimSpace(config.SecretID.ValueString())
	if secretID == "" {
		resp.Diagnostics.AddError("Missing secret_id", "`secret_id` must be supplied when opening `tensordock_secret_value`.")
		return
	}

	result, err := r.fetchSecretValue(ctx, secretID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to open TensorDock secret value", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &result)...)
}

func (r *SecretValueEphemeralResource) fetchSecretValue(ctx context.Context, secretID string) (SecretValueEphemeralModel, error) {
	secret, err := r.client.GetSecret(ctx, secretID)
	if err != nil {
		return SecretValueEphemeralModel{}, err
	}
	if strings.TrimSpace(secret.Value) == "" {
		return SecretValueEphemeralModel{}, fmt.Errorf("the selected TensorDock secret did not return a usable `value`")
	}

	return SecretValueEphemeralModel{
		SecretID: types.StringValue(secretID),
		ID:       types.StringValue(secret.ID),
		Name:     types.StringValue(secret.Name),
		Type:     types.StringValue(secret.Type),
		Value:    types.StringValue(secret.Value),
	}, nil
}
