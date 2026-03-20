package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &HostnodesDataSource{}

func NewHostnodesDataSource() datasource.DataSource {
	return &HostnodesDataSource{}
}

type HostnodesDataSource struct {
	client *Client
}

type HostnodesDataSourceModel struct {
	Hostnodes types.List `tfsdk:"hostnodes"`
}

func (d *HostnodesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hostnodes"
}

func (d *HostnodesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch all TensorDock hostnodes returned by the public API.",
		Attributes: map[string]schema.Attribute{
			"hostnodes": schema.ListNestedAttribute{
				MarkdownDescription: "Hostnodes returned by `GET /hostnodes`.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                schema.StringAttribute{Computed: true},
						"location_id":       schema.StringAttribute{Computed: true},
						"engine":            schema.StringAttribute{Computed: true},
						"uptime_percentage": schema.Float64Attribute{Computed: true},
						"available_resources": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"gpus": schema.ListNestedAttribute{
									Computed: true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"v0_name":         schema.StringAttribute{Computed: true},
											"available_count": schema.Int64Attribute{Computed: true},
											"price_per_hr":    schema.Float64Attribute{Computed: true},
										},
									},
								},
								"vcpu_count":              schema.Int64Attribute{Computed: true},
								"ram_gb":                  schema.Int64Attribute{Computed: true},
								"storage_gb":              schema.Int64Attribute{Computed: true},
								"max_vcpus_per_gpu":       schema.Int64Attribute{Computed: true},
								"max_ram_per_gpu":         schema.Int64Attribute{Computed: true},
								"max_vcpus":               schema.Int64Attribute{Computed: true},
								"max_ram_gb":              schema.Int64Attribute{Computed: true},
								"max_storage_gb":          schema.Int64Attribute{Computed: true},
								"available_ports":         schema.ListAttribute{Computed: true, ElementType: types.Int64Type},
								"has_public_ip_available": schema.BoolAttribute{Computed: true},
							},
						},
						"pricing": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"per_vcpu_hr":       schema.Float64Attribute{Computed: true},
								"per_gb_ram_hr":     schema.Float64Attribute{Computed: true},
								"per_gb_storage_hr": schema.Float64Attribute{Computed: true},
							},
						},
						"location": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"uuid":                      schema.StringAttribute{Computed: true},
								"city":                      schema.StringAttribute{Computed: true},
								"stateprovince":             schema.StringAttribute{Computed: true},
								"country":                   schema.StringAttribute{Computed: true},
								"has_network_storage":       schema.BoolAttribute{Computed: true},
								"network_speed_gbps":        schema.Int64Attribute{Computed: true},
								"network_speed_upload_gbps": schema.Int64Attribute{Computed: true},
								"organization":              schema.StringAttribute{Computed: true},
								"organization_name":         schema.StringAttribute{Computed: true},
								"tier":                      schema.Int64Attribute{Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

func (d *HostnodesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *HostnodesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	hostnodes, err := d.client.ListHostnodes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list TensorDock hostnodes", err.Error())
		return
	}

	hostnodeValues, diags := buildHostnodesValue(hostnodes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := HostnodesDataSourceModel{
		Hostnodes: hostnodeValues,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
