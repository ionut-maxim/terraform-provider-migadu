package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/ionut-maxim/migadu"
)

var _ datasource.DataSource = &IdentityDataSource{}

// IdentityDataSourceModel reuses IdentityModel — the write-only password field is excluded.
type IdentityDataSourceModel = IdentityModel

type IdentityDataSource struct {
	client *migadu.Client
}

func NewIdentityDataSource() datasource.DataSource {
	return &IdentityDataSource{}
}

func (d *IdentityDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (d *IdentityDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a Migadu mailbox identity.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain this identity belongs to.",
			},
			"mailbox_local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of the parent mailbox.",
			},
			"local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of this identity.",
			},
			"address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full identity address.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The display name of this identity.",
			},
			"may_send": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this identity may send.",
			},
			"may_receive": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this identity may receive.",
			},
			"may_access_imap": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether IMAP access is allowed.",
			},
			"may_access_pop3": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether POP3 access is allowed.",
			},
			"may_access_managesieve": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether ManageSieve access is allowed.",
			},
			"footer_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the email footer is active.",
			},
			"footer_plain_body": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Plain-text footer body.",
			},
			"footer_html_body": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "HTML footer body.",
			},
		},
	}
}

func (d *IdentityDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdentityDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identity, err := d.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Identities().Identity(data.LocalPart.ValueString()).Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading identity", err.Error())
		return
	}

	identityToModel(identity, data.DomainName.ValueString(), data.MailboxLocalPart.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
