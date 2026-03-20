package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultBaseURL = "https://dashboard.tensordock.com/api/v2"

var _ provider.Provider = &TensorDockProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TensorDockProvider{version: version}
	}
}

type TensorDockProvider struct {
	version string
}

type TensorDockProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
	BaseURL  types.String `tfsdk:"base_url"`
}

func (p *TensorDockProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tensordock"
	resp.Version = p.version
}

func (p *TensorDockProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for provisioning TensorDock instances through the public v2 REST API.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "TensorDock API token. Can also be supplied with the `TENSORDOCK_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("TensorDock API base URL. Defaults to `%s`. Can also be supplied with `TENSORDOCK_BASE_URL`.", defaultBaseURL),
				Optional:            true,
			},
		},
	}
}

func (p *TensorDockProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data TensorDockProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiToken := strings.TrimSpace(os.Getenv("TENSORDOCK_API_TOKEN"))
	if !data.APIToken.IsNull() && !data.APIToken.IsUnknown() {
		apiToken = strings.TrimSpace(data.APIToken.ValueString())
	}

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing TensorDock API token",
			"Set `api_token` in provider configuration or export `TENSORDOCK_API_TOKEN`.",
		)
		return
	}

	baseURL := strings.TrimSpace(os.Getenv("TENSORDOCK_BASE_URL"))
	if !data.BaseURL.IsNull() && !data.BaseURL.IsUnknown() {
		baseURL = strings.TrimSpace(data.BaseURL.ValueString())
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	client, err := NewClient(baseURL, apiToken, p.version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid provider configuration",
			err.Error(),
		)
		return
	}

	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *TensorDockProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
		NewSecretResource,
	}
}

func (p *TensorDockProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHostnodesDataSource,
		NewLocationsDataSource,
	}
}
