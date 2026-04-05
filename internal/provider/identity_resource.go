package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ resource.Resource = &IdentityResource{}
var _ resource.ResourceWithImportState = &IdentityResource{}

type IdentityResource struct {
	client *migadu.Client
}

// IdentityModel holds the fields returned by the Migadu API, shared with the data source.
type IdentityModel struct {
	DomainName           types.String `tfsdk:"domain_name"`
	MailboxLocalPart     types.String `tfsdk:"mailbox_local_part"`
	LocalPart            types.String `tfsdk:"local_part"`
	Address              types.String `tfsdk:"address"`
	Name                 types.String `tfsdk:"name"`
	MaySend              types.Bool   `tfsdk:"may_send"`
	MayReceive           types.Bool   `tfsdk:"may_receive"`
	MayAccessImap        types.Bool   `tfsdk:"may_access_imap"`
	MayAccessPop3        types.Bool   `tfsdk:"may_access_pop3"`
	MayAccessManagesieve types.Bool   `tfsdk:"may_access_managesieve"`
	FooterActive         types.Bool   `tfsdk:"footer_active"`
	FooterPlainBody      types.String `tfsdk:"footer_plain_body"`
	FooterHtmlBody       types.String `tfsdk:"footer_html_body"`
}

type IdentityResourceModel struct {
	IdentityModel
	// Write-only field not returned by the API.
	Password types.String `tfsdk:"password"`
}

func NewIdentityResource() resource.Resource {
	return &IdentityResource{}
}

func (r *IdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (r *IdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Migadu mailbox identity.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain this identity belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"mailbox_local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of the parent mailbox.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of this identity.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full identity address.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name of this identity.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The identity password. Not read back from the API.",
			},
			"may_send": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this identity may send (inherited from mailbox).",
			},
			"may_receive": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this identity may receive (inherited from mailbox).",
			},
			"may_access_imap": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether IMAP access is allowed (inherited from mailbox).",
			},
			"may_access_pop3": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether POP3 access is allowed (inherited from mailbox).",
			},
			"may_access_managesieve": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether ManageSieve access is allowed (inherited from mailbox).",
			},
			"footer_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the email footer is active (inherited from mailbox).",
			},
			"footer_plain_body": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Plain-text footer body (inherited from mailbox).",
			},
			"footer_html_body": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "HTML footer body (inherited from mailbox).",
			},
		},
	}
}

func (r *IdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IdentityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.CreateIdentityRequest{
		Name:      data.Name.ValueString(),
		LocalPart: data.LocalPart.ValueString(),
	}
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		request.Password = data.Password.ValueString()
	}

	identity, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Identities().Create(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating identity", err.Error())
		return
	}

	identityToModel(identity, data.DomainName.ValueString(), data.MailboxLocalPart.ValueString(), &data.IdentityModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identity, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Identities().Identity(data.LocalPart.ValueString()).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading identity", err.Error())
		return
	}

	identityToModel(identity, data.DomainName.ValueString(), data.MailboxLocalPart.ValueString(), &data.IdentityModel)
	// password is not returned by the API; preserve state value.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IdentityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.UpdateIdentityRequest{
		Name: data.Name.ValueString(),
	}
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		request.Password = data.Password.ValueString()
	}

	identity, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Identities().Identity(data.LocalPart.ValueString()).Update(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error updating identity", err.Error())
		return
	}

	identityToModel(identity, data.DomainName.ValueString(), data.MailboxLocalPart.ValueString(), &data.IdentityModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Identities().Identity(data.LocalPart.ValueString()).Delete(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting identity", err.Error())
	}
}

func (r *IdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: <domain_name>/<mailbox_local_part>/<local_part>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mailbox_local_part"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("local_part"), parts[2])...)
}

func identityToModel(id migadu.Identity, domainName, mailboxLocalPart string, m *IdentityModel) {
	m.DomainName = types.StringValue(domainName)
	m.MailboxLocalPart = types.StringValue(mailboxLocalPart)
	m.LocalPart = types.StringValue(id.LocalPart)
	m.Address = types.StringValue(id.Address)
	m.Name = types.StringValue(id.Name)
	m.MaySend = types.BoolValue(id.MaySend)
	m.MayReceive = types.BoolValue(id.MayReceive)
	m.MayAccessImap = types.BoolValue(id.MayAccessImap)
	m.MayAccessPop3 = types.BoolValue(id.MayAccessPop3)
	m.MayAccessManagesieve = types.BoolValue(id.MayAccessManagesieve)
	m.FooterActive = types.BoolValue(id.FooterActive)
	m.FooterPlainBody = types.StringValue(id.FooterPlainBody)
	m.FooterHtmlBody = types.StringValue(id.FooterHtmlBody)
}
