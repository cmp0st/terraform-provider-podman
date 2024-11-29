// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/containers/podman/v5/pkg/bindings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &PodmanProvider{}
var _ provider.ProviderWithFunctions = &PodmanProvider{}

// Provider defines the provider implementation.
type PodmanProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type PodmanProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
}

func (p *PodmanProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "podman"
	resp.Version = p.version
}

func (p *PodmanProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            true,
			},
		},
	}
}

func (p *PodmanProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data PodmanProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	var endpoint string
	if data.Endpoint.IsNull() {
		dir, set := os.LookupEnv("XDG_RUNTIME_DIR")
		if !set {
			resp.Diagnostics.AddError("default endpoint cannot be used", "XDG_RUNTIME_DIR env var isn't set")
			return
		}
		endpoint = fmt.Sprintf(`unix:%s/podman/podman.sock`, dir)
	} else {
		endpoint = data.Endpoint.ValueString()
	}

	conn, err := bindings.NewConnection(context.Background(), endpoint)
	if err != nil {
		resp.Diagnostics.AddError("failed to connect to podman socket", err.Error())
		return
	}
	resp.ResourceData = conn
	resp.DataSourceData = conn
}

func (p *PodmanProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSecretResource,
	}
}

func (p *PodmanProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func (p *PodmanProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PodmanProvider{
			version: version,
		}
	}
}
