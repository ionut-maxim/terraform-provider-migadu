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

var _ resource.Resource = &ForwardingResource{}
var _ resource.ResourceWithImportState = &ForwardingResource{}

type ForwardingResource struct {
	client *migadu.Client
}

// ForwardingModel is shared between resource and data source.
type ForwardingModel struct {
	DomainName         types.String `tfsdk:"domain_name"`
	MailboxLocalPart   types.String `tfsdk:"mailbox_local_part"`
	Address            types.String `tfsdk:"address"`
	IsActive           types.Bool   `tfsdk:"is_active"`
	ExpiresOn          types.String `tfsdk:"expires_on"`
	RemoveUponExpiry   types.Bool   `tfsdk:"remove_upon_expiry"`
	BlockedAt          types.String `tfsdk:"blocked_at"`
	ConfirmationSentAt types.String `tfsdk:"confirmation_sent_at"`
	ConfirmedAt        types.String `tfsdk:"confirmed_at"`
}

func NewForwardingResource() resource.Resource {
	return &ForwardingResource{}
}

func (r *ForwardingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_forwarding"
}

func (r *ForwardingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Migadu mailbox forwarding.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain of the mailbox.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"mailbox_local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of the mailbox.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"address": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The forwarding destination address.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the forwarding is active.",
			},
			"expires_on": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When the forwarding expires.",
			},
			"remove_upon_expiry": schema.BoolAttribute{
				Optional:            true,
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

func (r *ForwardingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ForwardingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ForwardingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	forwarding, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Forwardings().Create(ctx, migadu.CreateForwardingRequest{
		Address: data.Address.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating forwarding", err.Error())
		return
	}

	// Apply any optional update fields (is_active, expires_on, remove_upon_expiry).
	updateReq := migadu.UpdateForwardingRequest{}
	hasUpdate := false
	if !data.IsActive.IsNull() && !data.IsActive.IsUnknown() {
		updateReq.IsActive = new(data.IsActive.ValueBool())
		hasUpdate = true
	}
	if !data.ExpiresOn.IsNull() && !data.ExpiresOn.IsUnknown() {
		updateReq.ExpiresOn = data.ExpiresOn.ValueString()
		hasUpdate = true
	}
	if !data.RemoveUponExpiry.IsNull() && !data.RemoveUponExpiry.IsUnknown() {
		updateReq.RemoveUponExpiry = new(data.RemoveUponExpiry.ValueBool())
		hasUpdate = true
	}

	if hasUpdate {
		forwarding, err = r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Forwardings().Forwarding(data.Address.ValueString()).Update(ctx, updateReq)
		if err != nil {
			resp.Diagnostics.AddError("Error configuring forwarding after creation", err.Error())
			return
		}
	}

	forwardingToModel(forwarding, data.DomainName.ValueString(), data.MailboxLocalPart.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ForwardingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ForwardingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	forwarding, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Forwardings().Forwarding(data.Address.ValueString()).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading forwarding", err.Error())
		return
	}

	forwardingToModel(forwarding, data.DomainName.ValueString(), data.MailboxLocalPart.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ForwardingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ForwardingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.UpdateForwardingRequest{}
	if !data.IsActive.IsNull() && !data.IsActive.IsUnknown() {
		request.IsActive = new(data.IsActive.ValueBool())
	}
	if !data.ExpiresOn.IsNull() && !data.ExpiresOn.IsUnknown() {
		request.ExpiresOn = data.ExpiresOn.ValueString()
	}
	if !data.RemoveUponExpiry.IsNull() && !data.RemoveUponExpiry.IsUnknown() {
		request.RemoveUponExpiry = new(data.RemoveUponExpiry.ValueBool())
	}

	forwarding, err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Forwardings().Forwarding(data.Address.ValueString()).Update(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error updating forwarding", err.Error())
		return
	}

	forwardingToModel(forwarding, data.DomainName.ValueString(), data.MailboxLocalPart.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ForwardingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ForwardingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Domains().Domain(data.DomainName.ValueString()).Mailboxes().Mailbox(data.MailboxLocalPart.ValueString()).Forwardings().Forwarding(data.Address.ValueString()).Delete(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting forwarding", err.Error())
	}
}

func (r *ForwardingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: <domain_name>/<mailbox_local_part>/<address>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mailbox_local_part"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("address"), parts[2])...)
}

func forwardingToModel(f migadu.Forwarding, domainName, mailboxLocalPart string, m *ForwardingModel) {
	m.DomainName = types.StringValue(domainName)
	m.MailboxLocalPart = types.StringValue(mailboxLocalPart)
	m.Address = types.StringValue(f.Address)
	m.IsActive = types.BoolValue(f.IsActive)
	m.ExpiresOn = types.StringValue(f.ExpiresOn)
	m.RemoveUponExpiry = types.BoolValue(f.RemoveUponExpiry)
	m.BlockedAt = types.StringValue(f.BlockedAt)
	m.ConfirmationSentAt = types.StringValue(f.ConfirmationSentAt)
	m.ConfirmedAt = types.StringValue(f.ConfirmedAt)
}
