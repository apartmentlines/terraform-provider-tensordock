package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &LocationsDataSource{}

func NewLocationsDataSource() datasource.DataSource {
	return &LocationsDataSource{}
}

type LocationsDataSource struct {
	client *Client
}

type LocationsDataSourceModel struct {
	Locations types.List `tfsdk:"locations"`
}

func (d *LocationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

func (d *LocationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch all TensorDock locations returned by the public API.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.ListNestedAttribute{
				MarkdownDescription: "Locations returned by `GET /locations`.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true},
						"city":          schema.StringAttribute{Computed: true},
						"stateprovince": schema.StringAttribute{Computed: true},
						"country":       schema.StringAttribute{Computed: true},
						"tier":          schema.Int64Attribute{Computed: true},
						"gpus": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"v0_name":      schema.StringAttribute{Computed: true},
									"display_name": schema.StringAttribute{Computed: true},
									"max_count":    schema.Int64Attribute{Computed: true},
									"price_per_hr": schema.Float64Attribute{Computed: true},
									"resources": schema.SingleNestedAttribute{
										Computed: true,
										Attributes: map[string]schema.Attribute{
											"max_vcpus":      schema.Int64Attribute{Computed: true},
											"max_ram_gb":     schema.Int64Attribute{Computed: true},
											"max_storage_gb": schema.Int64Attribute{Computed: true},
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
									"network_features": schema.SingleNestedAttribute{
										Computed: true,
										Attributes: map[string]schema.Attribute{
											"dedicated_ip_available":    schema.BoolAttribute{Computed: true},
											"port_forwarding_available": schema.BoolAttribute{Computed: true},
											"network_storage_available": schema.BoolAttribute{Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *LocationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LocationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	locations, err := d.client.ListLocations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list TensorDock locations", err.Error())
		return
	}

	locationValues, diags := buildLocationsValue(locations)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := LocationsDataSourceModel{
		Locations: locationValues,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
