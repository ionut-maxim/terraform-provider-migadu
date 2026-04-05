package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ datasource.DataSource = &RewriteDataSource{}

// RewriteDataSourceModel reuses RewriteModel — all fields are identical to the resource.
type RewriteDataSourceModel = RewriteModel

type RewriteDataSource struct {
	client *migadu.Client
}

func NewRewriteDataSource() datasource.DataSource {
	return &RewriteDataSource{}
}

func (d *RewriteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rewrite"
}

func (d *RewriteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a Migadu rewrite rule.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain this rewrite rule belongs to.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name (identifier) of the rewrite rule.",
			},
			"local_part_rule": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The local part rule pattern.",
			},
			"order_num": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The ordering number of the rule.",
			},
			"destinations": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of destination addresses.",
			},
		},
	}
}

func (d *RewriteDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RewriteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RewriteDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rewrite, err := d.client.Domains().Domain(data.DomainName.ValueString()).Rewrites().Rewrite(data.Name.ValueString()).Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading rewrite", err.Error())
		return
	}

	rewriteToModel(rewrite, data.DomainName.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
