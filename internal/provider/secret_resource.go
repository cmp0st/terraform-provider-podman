// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/containers/podman/v5/pkg/bindings/secrets"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SecretResource{}
var _ resource.ResourceWithImportState = &SecretResource{}

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

// SecretResource defines the resource implementation.
type SecretResource struct {
	conn context.Context
}

// SecretResourceModel describes the resource data model.
type SecretResourceModel struct {
	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Driver     types.String `tfsdk:"driver"`
	DriverOpts types.Map    `tfsdk:"driver_opts"`
	Labels     types.Map    `tfsdk:"labels"`
	Secret     types.String `tfsdk:"secret"`
}

func (r *SecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Secret",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"driver": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"labels": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"driver_opts": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"secret": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	conn, ok := req.ProviderData.(context.Context)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.conn = conn
}

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	driverOpts := map[string]string{}
	labels := map[string]string{}

	resp.Diagnostics.Append(data.DriverOpts.ElementsAs(ctx, &driverOpts, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := secrets.Create(r.conn, strings.NewReader(data.Secret.ValueString()), &secrets.CreateOptions{
		Name:       data.Name.ValueStringPointer(),
		Driver:     data.Driver.ValueStringPointer(),
		DriverOpts: driverOpts,
		Labels:     labels,
	})
	if err != nil {
		resp.Diagnostics.AddError("failed to add secret", err.Error())
		return
	}
	data.Id = types.StringValue(createResp.ID)

	if data.Driver.IsNull() {
		data.Driver = types.StringValue("file")
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := secrets.List(r.conn, &secrets.ListOptions{
		Filters: map[string][]string{
			"id": {data.Id.ValueString()},
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("failed to get secret", err.Error())
		return
	}

	if len(secret) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = basetypes.NewStringValue(secret[0].Spec.Name)
	data.Driver = basetypes.NewStringValue(secret[0].Spec.Driver.Name)

	driverOpts, diag := basetypes.NewMapValueFrom(ctx, types.StringType, secret[0].Spec.Driver.Options)
	resp.Diagnostics.Append(diag...)
	data.DriverOpts = driverOpts

	labels, diag := basetypes.NewMapValueFrom(ctx, types.StringType, secret[0].Spec.Labels)
	resp.Diagnostics.Append(diag...)
	data.Labels = labels

	data.Secret = basetypes.NewStringValue(secret[0].SecretData)
	tflog.Error(ctx, fmt.Sprintf("secretdata: %q", secret[0].SecretData))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SecretResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecretResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := secrets.Remove(r.conn, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("failed to delete secret", err.Error())
		return
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
