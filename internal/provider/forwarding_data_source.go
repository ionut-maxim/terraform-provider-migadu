package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/ionut-maxim/migadu"
)

var _ datasource.DataSource = &ForwardingDataSource{}

// ForwardingDataSourceModel reuses ForwardingModel — all fields are identical to the resource.
type ForwardingDataSourceModel = ForwardingModel

type ForwardingDataSource struct {
	client *migadu.Client
}

func NewForwardingDataSource() datasource.DataSource {
	return &ForwardingDataSource{}
}

func (d *ForwardingDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_forwarding"
}

func (d *ForwardingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a Migadu mailbox forwarding.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain of the mailbox.",
			},
			"mailbox_local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of the mailbox.",
			},
			"address": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The forwarding destination address.",
			},
			"is_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the forwarding is active.",
			},
			"expires_on": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the forwarding expires.",
			},
			"remove_upon_expiry": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether to remove the forwarding upon expiry.",
			},
			"blocked_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the forwarding was blocked.",
			},
			"confirmation_sent_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the confirmation was sent.",
			},
			"confirmed_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the forwarding was confirmed.",
			},
		},
	}
}

func (d *ForwardingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ForwardingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ForwardingDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	forwarding, err := d.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Forwardings().Forwarding(data.Address.ValueString()).Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading forwarding", err.Error())
		return
	}

	forwardingToModel(forwarding, data.DomainName.ValueString(), data.MailboxLocalPart.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
