package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSecretResourceSchemaUsesWriteOnlyVersionPattern(t *testing.T) {
	secretResource := &SecretResource{}
	var resp resource.SchemaResponse

	secretResource.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	valueAttr, ok := resp.Schema.Attributes["value_wo"]
	if !ok {
		t.Fatal("expected value_wo attribute in schema")
	}

	valueStringAttr, ok := valueAttr.(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected value_wo to be a StringAttribute, got %T", valueAttr)
	}
	if !valueStringAttr.WriteOnly {
		t.Fatal("expected value_wo to be write-only")
	}

	versionAttr, ok := resp.Schema.Attributes["value_wo_version"]
	if !ok {
		t.Fatal("expected value_wo_version attribute in schema")
	}

	versionIntAttr, ok := versionAttr.(resourceschema.Int64Attribute)
	if !ok {
		t.Fatalf("expected value_wo_version to be an Int64Attribute, got %T", versionAttr)
	}
	if !versionIntAttr.Optional || !versionIntAttr.Computed {
		t.Fatal("expected value_wo_version to be optional+computed")
	}

	if _, exists := resp.Schema.Attributes["value"]; exists {
		t.Fatal("did not expect legacy value attribute in schema")
	}
}

func TestValidateSecretPlanRequiresCreateInputs(t *testing.T) {
	plan := SecretResourceModel{
		Name:           types.StringNull(),
		Type:           types.StringNull(),
		ValueWOVersion: types.Int64Value(1),
	}

	diags := validateSecretPlan(plan, "")
	if !diags.HasError() {
		t.Fatal("expected diagnostics for missing secret inputs")
	}
}

func TestValidateSecretPlanAcceptsCompleteInput(t *testing.T) {
	plan := SecretResourceModel{
		Name:           types.StringValue("deploy-key"),
		Type:           types.StringValue("SSHKEY"),
		ValueWOVersion: types.Int64Value(1),
	}

	diags := validateSecretPlan(plan, "ssh-ed25519 AAAA...")
	if diags.HasError() {
		t.Fatalf("expected no diagnostics, got: %+v", diags)
	}
}

func TestNormalizeSecretCreatePlanDefaultsVersionToOne(t *testing.T) {
	plan := SecretResourceModel{
		Name:           types.StringValue("deploy-key"),
		Type:           types.StringValue("SSHKEY"),
		ValueWOVersion: types.Int64Null(),
	}

	got := normalizeSecretCreatePlan(plan, types.StringValue("ssh-ed25519 AAAA..."))
	if got.ValueWOVersion.IsNull() || got.ValueWOVersion.ValueInt64() != 1 {
		t.Fatalf("expected create plan version default to 1, got: %#v", got.ValueWOVersion)
	}
}

func TestNormalizeSecretCreatePlanPreservesExplicitVersion(t *testing.T) {
	plan := SecretResourceModel{
		Name:           types.StringValue("deploy-key"),
		Type:           types.StringValue("SSHKEY"),
		ValueWOVersion: types.Int64Value(2),
	}

	got := normalizeSecretCreatePlan(plan, types.StringValue("ssh-ed25519 BBBB..."))
	if got.ValueWOVersion.ValueInt64() != 2 {
		t.Fatalf("expected explicit create version to be preserved, got: %#v", got.ValueWOVersion)
	}
}

func TestRequiresReplaceSecretValueRotationOnlyWhenRotatingWithValue(t *testing.T) {
	plan := SecretResourceModel{
		ID:             types.StringValue("secret-1"),
		ValueWOVersion: types.Int64Value(2),
	}
	state := SecretResourceModel{
		ID:             types.StringValue("secret-1"),
		ValueWOVersion: types.Int64Null(),
	}

	if got := requiresReplaceSecretValueRotation(plan, state, types.StringNull()); got != nil {
		t.Fatalf("expected no replacement paths without value_wo, got: %#v", got)
	}

	got := requiresReplaceSecretValueRotation(plan, state, types.StringValue("ssh-ed25519 AAAA..."))
	if len(got) != 1 || !got[0].Equal(path.Root("value_wo_version")) {
		t.Fatalf("expected replacement on value_wo_version when rotating, got: %#v", got)
	}
}
