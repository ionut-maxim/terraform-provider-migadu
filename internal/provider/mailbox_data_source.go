package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ datasource.DataSource = &MailboxDataSource{}

// MailboxDataSourceModel reuses MailboxModel — write-only fields (password, etc.) are excluded.
type MailboxDataSourceModel = MailboxModel

type MailboxDataSource struct {
	client *migadu.Client
}

func NewMailboxDataSource() datasource.DataSource {
	return &MailboxDataSource{}
}

func (d *MailboxDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mailbox"
}

func (d *MailboxDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a Migadu mailbox.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain this mailbox belongs to.",
			},
			"local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of the mailbox address.",
			},
			"address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full mailbox address.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The display name of the mailbox.",
			},
			"is_internal": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the mailbox is internal.",
			},
			"wildcard_sender": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether wildcard sender is enabled.",
			},
			"may_send": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the mailbox may send.",
			},
			"may_receive": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the mailbox may receive.",
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
			"password_recovery_email": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Email address used for password recovery.",
			},
			"spam_action": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Action to take on spam messages.",
			},
			"spam_aggressiveness": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Spam aggressiveness level.",
			},
			"sender_denylist": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of denied senders.",
			},
			"sender_allowlist": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of allowed senders.",
			},
			"recipient_denylist": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of denied recipients.",
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

func (d *MailboxDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MailboxDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MailboxDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mailbox, err := d.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.LocalPart.ValueString()).Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading mailbox", err.Error())
		return
	}

	mailboxToModel(mailbox, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
