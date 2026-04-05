package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ datasource.DataSource = &AliasDataSource{}

// AliasDataSourceModel reuses AliasModel — all fields are identical to the resource.
type AliasDataSourceModel = AliasModel

type AliasDataSource struct {
	client *migadu.Client
}

func NewAliasDataSource() datasource.DataSource {
	return &AliasDataSource{}
}

func (d *AliasDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alias"
}

func (d *AliasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a Migadu alias.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain this alias belongs to.",
			},
			"local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of the alias address.",
			},
			"address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full alias address.",
			},
			"is_internal": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the alias is internal.",
			},
			"destinations": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of destination addresses.",
			},
		},
	}
}

func (d *AliasDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*migadu.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *migadu.Client, got %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *AliasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AliasDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alias, err := d.client.Domains().Domain(data.DomainName.ValueString()).Aliases().Alias(data.LocalPart.ValueString()).Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading alias", err.Error())
		return
	}

	aliasToModel(alias, data.DomainName.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
