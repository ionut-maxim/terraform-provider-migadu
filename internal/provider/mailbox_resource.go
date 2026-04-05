package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ resource.Resource = &MailboxResource{}
var _ resource.ResourceWithImportState = &MailboxResource{}

type MailboxResource struct {
	client *migadu.Client
}

// MailboxModel holds the fields returned by the Migadu API, shared with the data source.
type MailboxModel struct {
	DomainName            types.String `tfsdk:"domain_name"`
	LocalPart             types.String `tfsdk:"local_part"`
	Address               types.String `tfsdk:"address"`
	Name                  types.String `tfsdk:"name"`
	IsInternal            types.Bool   `tfsdk:"is_internal"`
	WildcardSender        types.Bool   `tfsdk:"wildcard_sender"`
	MaySend               types.Bool   `tfsdk:"may_send"`
	MayReceive            types.Bool   `tfsdk:"may_receive"`
	MayAccessImap         types.Bool   `tfsdk:"may_access_imap"`
	MayAccessPop3         types.Bool   `tfsdk:"may_access_pop3"`
	MayAccessManagesieve  types.Bool   `tfsdk:"may_access_managesieve"`
	PasswordRecoveryEmail types.String `tfsdk:"password_recovery_email"`
	SpamAction            types.String `tfsdk:"spam_action"`
	SpamAggressiveness    types.String `tfsdk:"spam_aggressiveness"`
	SenderDenylist        types.List   `tfsdk:"sender_denylist"`
	SenderAllowlist       types.List   `tfsdk:"sender_allowlist"`
	RecipientDenylist     types.List   `tfsdk:"recipient_denylist"`
	FooterActive          types.Bool   `tfsdk:"footer_active"`
	FooterPlainBody       types.String `tfsdk:"footer_plain_body"`
	FooterHtmlBody        types.String `tfsdk:"footer_html_body"`
}

type MailboxResourceModel struct {
	MailboxModel
	// Write-only fields not returned by the API.
	Password       types.String `tfsdk:"password"`
	PasswordMethod types.String `tfsdk:"password_method"`
	ForwardingTo   types.String `tfsdk:"forwarding_to"`
}

func NewMailboxResource() resource.Resource {
	return &MailboxResource{}
}

func (r *MailboxResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mailbox"
}

func (r *MailboxResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Migadu mailbox.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain this mailbox belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of the mailbox address.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full mailbox address.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name of the mailbox.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The mailbox password. Not read back from the API.",
			},
			"password_method": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Password method used on creation (e.g. 'password' or 'invitation'). Has no effect after creation.",
			},
			"password_recovery_email": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Email address used for password recovery.",
			},
			"forwarding_to": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Forward incoming messages to this address. Not read back from the API.",
			},
			"is_internal": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the mailbox is internal.",
			},
			"wildcard_sender": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether wildcard sender is enabled.",
			},
			"may_send": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the mailbox may send.",
			},
			"may_receive": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the mailbox may receive.",
			},
			"may_access_imap": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether IMAP access is allowed.",
			},
			"may_access_pop3": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether POP3 access is allowed.",
			},
			"may_access_managesieve": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether ManageSieve access is allowed.",
			},
			"spam_action": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Action to take on spam messages.",
			},
			"spam_aggressiveness": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Spam aggressiveness level.",
			},
			"sender_denylist": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of denied senders.",
			},
			"sender_allowlist": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of allowed senders.",
			},
			"recipient_denylist": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of denied recipients.",
			},
			"footer_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the email footer is active.",
			},
			"footer_plain_body": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Plain-text footer body.",
			},
			"footer_html_body": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "HTML footer body.",
			},
		},
	}
}

func (r *MailboxResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*migadu.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *migadu.Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *MailboxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MailboxResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := migadu.CreateMailboxRequest{
		Name:      data.Name.ValueString(),
		LocalPart: data.LocalPart.ValueString(),
	}
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		createReq.Password = data.Password.ValueString()
	}
	if !data.PasswordMethod.IsNull() && !data.PasswordMethod.IsUnknown() {
		createReq.PasswordMethod = data.PasswordMethod.ValueString()
	}
	if !data.PasswordRecoveryEmail.IsNull() && !data.PasswordRecoveryEmail.IsUnknown() {
		createReq.PasswordRecoveryEmail = data.PasswordRecoveryEmail.ValueString()
	}
	if !data.ForwardingTo.IsNull() && !data.ForwardingTo.IsUnknown() {
		createReq.ForwardingTo = data.ForwardingTo.ValueString()
	}
	if !data.IsInternal.IsNull() && !data.IsInternal.IsUnknown() {
		createReq.IsInternal = new(data.IsInternal.ValueBool())
	}

	_, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating mailbox", err.Error())
		return
	}

	// Apply update-only fields immediately after creation.
	updateReq, diags := buildMailboxUpdateRequest(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mailbox, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.LocalPart.ValueString()).Update(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring mailbox after creation", err.Error())
		return
	}

	mailboxToModel(mailbox, &data.MailboxModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MailboxResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MailboxResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mailbox, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.LocalPart.ValueString()).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading mailbox", err.Error())
		return
	}

	mailboxToModel(mailbox, &data.MailboxModel)
	// password, password_method and forwarding_to are not returned by the API; preserve state values.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MailboxResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MailboxResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := buildMailboxUpdateRequest(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mailbox, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.LocalPart.ValueString()).Update(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating mailbox", err.Error())
		return
	}

	mailboxToModel(mailbox, &data.MailboxModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MailboxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MailboxResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.LocalPart.ValueString()).Delete(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting mailbox", err.Error())
	}
}

func (r *MailboxResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: <domain_name>/<local_part>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("local_part"), parts[1])...)
}

func buildMailboxUpdateRequest(ctx context.Context, data MailboxResourceModel) (migadu.UpdateMailboxRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := migadu.UpdateMailboxRequest{
		Name: data.Name.ValueString(),
	}
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		req.Password = data.Password.ValueString()
	}
	if !data.PasswordRecoveryEmail.IsNull() && !data.PasswordRecoveryEmail.IsUnknown() {
		req.PasswordRecoveryEmail = data.PasswordRecoveryEmail.ValueString()
	}
	if !data.ForwardingTo.IsNull() && !data.ForwardingTo.IsUnknown() {
		req.ForwardingTo = data.ForwardingTo.ValueString()
	}
	if !data.IsInternal.IsNull() && !data.IsInternal.IsUnknown() {
		req.IsInternal = new(data.IsInternal.ValueBool())
	}
	if !data.MaySend.IsNull() && !data.MaySend.IsUnknown() {
		req.MaySend = new(data.MaySend.ValueBool())
	}
	if !data.MayReceive.IsNull() && !data.MayReceive.IsUnknown() {
		req.MayReceive = new(data.MayReceive.ValueBool())
	}
	if !data.MayAccessImap.IsNull() && !data.MayAccessImap.IsUnknown() {
		req.MayAccessImap = new(data.MayAccessImap.ValueBool())
	}
	if !data.MayAccessPop3.IsNull() && !data.MayAccessPop3.IsUnknown() {
		req.MayAccessPop3 = new(data.MayAccessPop3.ValueBool())
	}
	if !data.MayAccessManagesieve.IsNull() && !data.MayAccessManagesieve.IsUnknown() {
		req.MayAccessManagesieve = new(data.MayAccessManagesieve.ValueBool())
	}
	if !data.SpamAction.IsNull() && !data.SpamAction.IsUnknown() {
		req.SpamAction = data.SpamAction.ValueString()
	}
	if !data.SpamAggressiveness.IsNull() && !data.SpamAggressiveness.IsUnknown() {
		req.SpamAggressiveness = data.SpamAggressiveness.ValueString()
	}
	if !data.SenderDenylist.IsNull() && !data.SenderDenylist.IsUnknown() {
		diags.Append(data.SenderDenylist.ElementsAs(ctx, &req.SenderDenylist, false)...)
	}
	if !data.SenderAllowlist.IsNull() && !data.SenderAllowlist.IsUnknown() {
		diags.Append(data.SenderAllowlist.ElementsAs(ctx, &req.SenderAllowlist, false)...)
	}
	if !data.RecipientDenylist.IsNull() && !data.RecipientDenylist.IsUnknown() {
		diags.Append(data.RecipientDenylist.ElementsAs(ctx, &req.RecipientDenylist, false)...)
	}
	if !data.FooterActive.IsNull() && !data.FooterActive.IsUnknown() {
		req.FooterActive = new(data.FooterActive.ValueBool())
	}
	if !data.FooterPlainBody.IsNull() && !data.FooterPlainBody.IsUnknown() {
		req.FooterPlainBody = data.FooterPlainBody.ValueString()
	}
	if !data.FooterHtmlBody.IsNull() && !data.FooterHtmlBody.IsUnknown() {
		req.FooterHtmlBody = data.FooterHtmlBody.ValueString()
	}
	return req, diags
}

func mailboxToModel(mb migadu.Mailbox, m *MailboxModel) {
	m.DomainName = types.StringValue(mb.DomainName)
	m.LocalPart = types.StringValue(mb.LocalPart)
	m.Address = types.StringValue(mb.Address)
	m.Name = types.StringValue(mb.Name)
	m.IsInternal = types.BoolValue(mb.IsInternal)
	m.WildcardSender = types.BoolValue(mb.WildcardSender)
	m.MaySend = types.BoolValue(mb.MaySend)
	m.MayReceive = types.BoolValue(mb.MayReceive)
	m.MayAccessImap = types.BoolValue(mb.MayAccessImap)
	m.MayAccessPop3 = types.BoolValue(mb.MayAccessPop3)
	m.MayAccessManagesieve = types.BoolValue(mb.MayAccessManagesieve)
	m.PasswordRecoveryEmail = types.StringValue(mb.PasswordRecoveryEmail)
	m.SpamAction = types.StringValue(mb.SpamAction)
	m.SpamAggressiveness = types.StringValue(mb.SpamAggressiveness)
	m.SenderDenylist = stringsToList(mb.SenderDenylist)
	m.SenderAllowlist = stringsToList(mb.SenderAllowlist)
	m.RecipientDenylist = stringsToList(mb.RecipientDenylist)
	m.FooterActive = types.BoolValue(mb.FooterActive)
	m.FooterPlainBody = types.StringValue(mb.FooterPlainBody)
	m.FooterHtmlBody = types.StringValue(mb.FooterHtmlBody)
}
