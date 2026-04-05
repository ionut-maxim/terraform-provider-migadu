package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ provider.Provider = &MigaduProvider{}

type MigaduProvider struct {
	version string
}

type MigaduProviderModel struct {
	Username types.String `tfsdk:"username"`
	APIKey   types.String `tfsdk:"api_key"`
}

func (m *MigaduProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "migadu"
	resp.Version = m.version
}

func (m *MigaduProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "Migadu username (email address used to log in).",
				Required:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Migadu API key.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (m *MigaduProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data MigaduProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := migadu.New(data.Username.ValueString(), data.APIKey.ValueString())
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (m *MigaduProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
		NewMailboxResource,
		NewAliasResource,
		NewRewriteResource,
		NewIdentityResource,
		NewForwardingResource,
	}
}

func (m *MigaduProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDomainDataSource,
		NewMailboxDataSource,
		NewAliasDataSource,
		NewRewriteDataSource,
		NewIdentityDataSource,
		NewForwardingDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MigaduProvider{
			version: version,
		}
	}
}
