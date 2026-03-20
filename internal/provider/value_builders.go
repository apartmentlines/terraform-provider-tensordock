package provider

import (
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	portForwardAttrTypes = map[string]attr.Type{
		"internal_port": types.Int64Type,
		"external_port": types.Int64Type,
	}
	portForwardObjectType = types.ObjectType{AttrTypes: portForwardAttrTypes}

	resourceLimitsAttrTypes = map[string]attr.Type{
		"max_vcpus":      types.Int64Type,
		"max_ram_gb":     types.Int64Type,
		"max_storage_gb": types.Int64Type,
	}
	resourceLimitsObjectType = types.ObjectType{AttrTypes: resourceLimitsAttrTypes}

	pricingAttrTypes = map[string]attr.Type{
		"per_vcpu_hr":       types.Float64Type,
		"per_gb_ram_hr":     types.Float64Type,
		"per_gb_storage_hr": types.Float64Type,
	}
	pricingObjectType = types.ObjectType{AttrTypes: pricingAttrTypes}

	networkFeaturesAttrTypes = map[string]attr.Type{
		"dedicated_ip_available":    types.BoolType,
		"port_forwarding_available": types.BoolType,
		"network_storage_available": types.BoolType,
	}
	networkFeaturesObjectType = types.ObjectType{AttrTypes: networkFeaturesAttrTypes}

	locationGPUAttrTypes = map[string]attr.Type{
		"v0_name":          types.StringType,
		"display_name":     types.StringType,
		"max_count":        types.Int64Type,
		"price_per_hr":     types.Float64Type,
		"resources":        resourceLimitsObjectType,
		"pricing":          pricingObjectType,
		"network_features": networkFeaturesObjectType,
	}
	locationGPUObjectType = types.ObjectType{AttrTypes: locationGPUAttrTypes}

	locationAttrTypes = map[string]attr.Type{
		"id":            types.StringType,
		"city":          types.StringType,
		"stateprovince": types.StringType,
		"country":       types.StringType,
		"tier":          types.Int64Type,
		"gpus":          types.ListType{ElemType: locationGPUObjectType},
	}
	locationObjectType = types.ObjectType{AttrTypes: locationAttrTypes}

	hostnodeGPUAttrTypes = map[string]attr.Type{
		"v0_name":         types.StringType,
		"available_count": types.Int64Type,
		"price_per_hr":    types.Float64Type,
	}
	hostnodeGPUObjectType = types.ObjectType{AttrTypes: hostnodeGPUAttrTypes}

	hostnodeAvailableResourcesAttrTypes = map[string]attr.Type{
		"gpus":                    types.ListType{ElemType: hostnodeGPUObjectType},
		"vcpu_count":              types.Int64Type,
		"ram_gb":                  types.Int64Type,
		"storage_gb":              types.Int64Type,
		"max_vcpus_per_gpu":       types.Int64Type,
		"max_ram_per_gpu":         types.Int64Type,
		"max_vcpus":               types.Int64Type,
		"max_ram_gb":              types.Int64Type,
		"max_storage_gb":          types.Int64Type,
		"available_ports":         types.ListType{ElemType: types.Int64Type},
		"has_public_ip_available": types.BoolType,
	}
	hostnodeAvailableResourcesObjectType = types.ObjectType{AttrTypes: hostnodeAvailableResourcesAttrTypes}

	hostnodeLocationAttrTypes = map[string]attr.Type{
		"uuid":                      types.StringType,
		"city":                      types.StringType,
		"stateprovince":             types.StringType,
		"country":                   types.StringType,
		"has_network_storage":       types.BoolType,
		"network_speed_gbps":        types.Float64Type,
		"network_speed_upload_gbps": types.Float64Type,
		"organization":              types.StringType,
		"organization_name":         types.StringType,
		"tier":                      types.Int64Type,
	}
	hostnodeLocationObjectType = types.ObjectType{AttrTypes: hostnodeLocationAttrTypes}

	hostnodeAttrTypes = map[string]attr.Type{
		"id":                  types.StringType,
		"location_id":         types.StringType,
		"engine":              types.StringType,
		"uptime_percentage":   types.Float64Type,
		"available_resources": hostnodeAvailableResourcesObjectType,
		"pricing":             pricingObjectType,
		"location":            hostnodeLocationObjectType,
	}
	hostnodeObjectType = types.ObjectType{AttrTypes: hostnodeAttrTypes}
)

func buildPortForwardList(portForwards []PortForward) types.List {
	sorted := append([]PortForward(nil), portForwards...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ExternalPort == sorted[j].ExternalPort {
			return sorted[i].InternalPort < sorted[j].InternalPort
		}
		return sorted[i].ExternalPort < sorted[j].ExternalPort
	})

	values := make([]attr.Value, 0, len(sorted))
	for _, portForward := range sorted {
		objectValue, diags := types.ObjectValue(portForwardAttrTypes, map[string]attr.Value{
			"internal_port": types.Int64Value(portForward.InternalPort),
			"external_port": types.Int64Value(portForward.ExternalPort),
		})
		if diags.HasError() {
			return types.ListNull(portForwardObjectType)
		}
		values = append(values, objectValue)
	}

	listValue, diags := types.ListValue(portForwardObjectType, values)
	if diags.HasError() {
		return types.ListNull(portForwardObjectType)
	}

	return listValue
}

func buildLocationsValue(locations []Location) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	values := make([]attr.Value, 0, len(locations))

	for _, location := range locations {
		gpusValue, gpuDiags := buildLocationGPUList(location.GPUs)
		diags.Append(gpuDiags...)

		locationValue, locationDiags := types.ObjectValue(locationAttrTypes, map[string]attr.Value{
			"id":            types.StringValue(location.ID),
			"city":          types.StringValue(location.City),
			"stateprovince": types.StringValue(location.StateProvince),
			"country":       types.StringValue(location.Country),
			"tier":          types.Int64Value(location.Tier),
			"gpus":          gpusValue,
		})
		diags.Append(locationDiags...)
		values = append(values, locationValue)
	}

	listValue, listDiags := types.ListValue(locationObjectType, values)
	diags.Append(listDiags...)
	return listValue, diags
}

func buildLocationGPUList(gpus []LocationGPU) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	values := make([]attr.Value, 0, len(gpus))

	for _, gpu := range gpus {
		resourcesValue, resourceDiags := types.ObjectValue(resourceLimitsAttrTypes, map[string]attr.Value{
			"max_vcpus":      types.Int64Value(gpu.Resources.MaxVCPUs),
			"max_ram_gb":     types.Int64Value(gpu.Resources.MaxRAMGB),
			"max_storage_gb": types.Int64Value(gpu.Resources.MaxStorageGB),
		})
		diags.Append(resourceDiags...)

		pricingValue, pricingDiags := buildPricingValue(gpu.Pricing)
		diags.Append(pricingDiags...)

		networkValue, networkDiags := types.ObjectValue(networkFeaturesAttrTypes, map[string]attr.Value{
			"dedicated_ip_available":    types.BoolValue(gpu.Network.DedicatedIPAvailable),
			"port_forwarding_available": types.BoolValue(gpu.Network.PortForwardingAvailble),
			"network_storage_available": types.BoolValue(gpu.Network.NetworkStorageAvailble),
		})
		diags.Append(networkDiags...)

		gpuValue, gpuDiags := types.ObjectValue(locationGPUAttrTypes, map[string]attr.Value{
			"v0_name":          types.StringValue(gpu.V0Name),
			"display_name":     types.StringValue(gpu.DisplayName),
			"max_count":        types.Int64Value(gpu.MaxCount),
			"price_per_hr":     types.Float64Value(gpu.PricePerHR),
			"resources":        resourcesValue,
			"pricing":          pricingValue,
			"network_features": networkValue,
		})
		diags.Append(gpuDiags...)
		values = append(values, gpuValue)
	}

	listValue, listDiags := types.ListValue(locationGPUObjectType, values)
	diags.Append(listDiags...)
	return listValue, diags
}

func buildHostnodesValue(hostnodes []Hostnode) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	values := make([]attr.Value, 0, len(hostnodes))

	for _, hostnode := range hostnodes {
		availableResourcesValue, resourcesDiags := buildHostnodeAvailableResourcesValue(hostnode.AvailableResources)
		diags.Append(resourcesDiags...)

		pricingValue, pricingDiags := buildPricingValue(hostnode.Pricing)
		diags.Append(pricingDiags...)

		locationValue, locationDiags := types.ObjectValue(hostnodeLocationAttrTypes, map[string]attr.Value{
			"uuid":                      types.StringValue(hostnode.Location.UUID),
			"city":                      types.StringValue(hostnode.Location.City),
			"stateprovince":             types.StringValue(hostnode.Location.StateProvince),
			"country":                   types.StringValue(hostnode.Location.Country),
			"has_network_storage":       types.BoolValue(hostnode.Location.HasNetworkStorage),
			"network_speed_gbps":        types.Float64Value(hostnode.Location.NetworkSpeedGbps),
			"network_speed_upload_gbps": types.Float64Value(hostnode.Location.NetworkSpeedUploadGbps),
			"organization":              types.StringValue(hostnode.Location.Organization),
			"organization_name":         types.StringValue(hostnode.Location.OrganizationName),
			"tier":                      types.Int64Value(hostnode.Location.Tier),
		})
		diags.Append(locationDiags...)

		hostnodeValue, hostnodeDiags := types.ObjectValue(hostnodeAttrTypes, map[string]attr.Value{
			"id":                  types.StringValue(hostnode.ID),
			"location_id":         types.StringValue(hostnode.LocationID),
			"engine":              types.StringValue(hostnode.Engine),
			"uptime_percentage":   types.Float64Value(hostnode.UptimePercentage),
			"available_resources": availableResourcesValue,
			"pricing":             pricingValue,
			"location":            locationValue,
		})
		diags.Append(hostnodeDiags...)
		values = append(values, hostnodeValue)
	}

	listValue, listDiags := types.ListValue(hostnodeObjectType, values)
	diags.Append(listDiags...)
	return listValue, diags
}

func buildHostnodeAvailableResourcesValue(resources HostnodeAvailableResources) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	gpusValue, gpuDiags := buildHostnodeGPUList(resources.GPUs)
	diags.Append(gpuDiags...)

	availablePortsValue, portDiags := buildInt64ListValue(resources.AvailablePorts)
	diags.Append(portDiags...)

	objectValue, objectDiags := types.ObjectValue(hostnodeAvailableResourcesAttrTypes, map[string]attr.Value{
		"gpus":                    gpusValue,
		"vcpu_count":              types.Int64Value(resources.VCPUCount),
		"ram_gb":                  types.Int64Value(resources.RAMGB),
		"storage_gb":              types.Int64Value(resources.StorageGB),
		"max_vcpus_per_gpu":       types.Int64Value(resources.MaxVCPUsPerGPU),
		"max_ram_per_gpu":         types.Int64Value(resources.MaxRAMPerGPU),
		"max_vcpus":               types.Int64Value(resources.MaxVCPUs),
		"max_ram_gb":              types.Int64Value(resources.MaxRAMGB),
		"max_storage_gb":          types.Int64Value(resources.MaxStorageGB),
		"available_ports":         availablePortsValue,
		"has_public_ip_available": types.BoolValue(resources.HasPublicIPAvailable),
	})
	diags.Append(objectDiags...)

	return objectValue, diags
}

func buildHostnodeGPUList(gpus []HostnodeGPU) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	values := make([]attr.Value, 0, len(gpus))

	for _, gpu := range gpus {
		gpuValue, gpuDiags := types.ObjectValue(hostnodeGPUAttrTypes, map[string]attr.Value{
			"v0_name":         types.StringValue(gpu.V0Name),
			"available_count": types.Int64Value(gpu.AvailableCount),
			"price_per_hr":    types.Float64Value(gpu.PricePerHR),
		})
		diags.Append(gpuDiags...)
		values = append(values, gpuValue)
	}

	listValue, listDiags := types.ListValue(hostnodeGPUObjectType, values)
	diags.Append(listDiags...)
	return listValue, diags
}

func buildInt64ListValue(values []int64) (types.List, diag.Diagnostics) {
	attrValues := make([]attr.Value, 0, len(values))
	for _, value := range values {
		attrValues = append(attrValues, types.Int64Value(value))
	}

	return types.ListValue(types.Int64Type, attrValues)
}

func buildPricingValue(pricing Pricing) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(pricingAttrTypes, map[string]attr.Value{
		"per_vcpu_hr":       types.Float64Value(pricing.PerVCPUHR),
		"per_gb_ram_hr":     types.Float64Value(pricing.PerGBRAMHR),
		"per_gb_storage_hr": types.Float64Value(pricing.PerGBStorageHR),
	})
}
