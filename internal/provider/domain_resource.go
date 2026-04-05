package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ resource.Resource = &DomainResource{}
var _ resource.ResourceWithImportState = &DomainResource{}

type DomainResource struct {
	client *migadu.Client
}

var dnsRecordAttrTypes = map[string]attr.Type{
	"name":  types.StringType,
	"type":  types.StringType,
	"value": types.StringType,
}

var mxRecordAttrTypes = map[string]attr.Type{
	"name":     types.StringType,
	"type":     types.StringType,
	"value":    types.StringType,
	"priority": types.Int64Type,
}

// DomainModel holds the fields shared between the domain resource and data source.
type DomainModel struct {
	Name                    types.String `tfsdk:"name"`
	HostedDNS               types.Bool   `tfsdk:"hosted_dns"`
	Description             types.String `tfsdk:"description"`
	Tags                    types.List   `tfsdk:"tags"`
	SpamAggressiveness      types.String `tfsdk:"spam_aggressiveness"`
	GreylistingEnabled      types.Bool   `tfsdk:"greylisting_enabled"`
	SubjectRewritingEnabled types.Bool   `tfsdk:"subject_rewriting_enabled"`
	JunkSubjectKeywordSpam  types.Bool   `tfsdk:"junk_subject_keyword_spam"`
	SenderDenylist          types.List   `tfsdk:"sender_denylist"`
	SenderAllowlist         types.List   `tfsdk:"sender_allowlist"`
	RecipientDenylist       types.List   `tfsdk:"recipient_denylist"`
	CatchallDestinations    types.List   `tfsdk:"catchall_destinations"`
	CanAccess               types.Bool   `tfsdk:"can_access"`
	MxProxyEnabled          types.Bool   `tfsdk:"mx_proxy_enabled"`
	State                   types.String `tfsdk:"state"`
	ActivatedAt             types.String `tfsdk:"activated_at"`
	DeactivatedAt           types.String `tfsdk:"deactivated_at"`
	CanReceive              types.Bool   `tfsdk:"can_receive"`
	CanSend                 types.Bool   `tfsdk:"can_send"`
	SPF                     types.Object `tfsdk:"spf"`
	DMARC                   types.Object `tfsdk:"dmarc"`
	DNSVerification         types.Object `tfsdk:"dns_verification"`
	DKIMRecords             types.List   `tfsdk:"dkim_records"`
	MXRecords               types.List   `tfsdk:"mx_records"`
}

type DomainResourceModel struct {
	DomainModel
	CreateDefaultAddresses types.Bool `tfsdk:"create_default_addresses"`
}

// domainMutableFields holds the optional fields shared by both Create and Update requests.
type domainMutableFields struct {
	Description             string
	Tags                    []string
	SpamAggressiveness      string
	GreylistingEnabled      *bool
	SubjectRewritingEnabled *bool
	JunkSubjectKeywordSpam  *bool
	SenderDenylist          []string
	SenderAllowlist         []string
	RecipientDenylist       []string
	CatchallDestinations    []string
	CanAccess               *bool
	MxProxyEnabled          *bool
}

func extractMutableFields(ctx context.Context, data DomainModel) (domainMutableFields, diag.Diagnostics) {
	var diags diag.Diagnostics
	f := domainMutableFields{}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		f.Description = data.Description.ValueString()
	}
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		diags.Append(data.Tags.ElementsAs(ctx, &f.Tags, false)...)
	}
	if !data.SpamAggressiveness.IsNull() && !data.SpamAggressiveness.IsUnknown() {
		f.SpamAggressiveness = data.SpamAggressiveness.ValueString()
	}
	if !data.GreylistingEnabled.IsNull() && !data.GreylistingEnabled.IsUnknown() {
		f.GreylistingEnabled = new(data.GreylistingEnabled.ValueBool())
	}
	if !data.SubjectRewritingEnabled.IsNull() && !data.SubjectRewritingEnabled.IsUnknown() {
		f.SubjectRewritingEnabled = new(data.SubjectRewritingEnabled.ValueBool())
	}
	if !data.JunkSubjectKeywordSpam.IsNull() && !data.JunkSubjectKeywordSpam.IsUnknown() {
		f.JunkSubjectKeywordSpam = new(data.JunkSubjectKeywordSpam.ValueBool())
	}
	if !data.SenderDenylist.IsNull() && !data.SenderDenylist.IsUnknown() {
		diags.Append(data.SenderDenylist.ElementsAs(ctx, &f.SenderDenylist, false)...)
	}
	if !data.SenderAllowlist.IsNull() && !data.SenderAllowlist.IsUnknown() {
		diags.Append(data.SenderAllowlist.ElementsAs(ctx, &f.SenderAllowlist, false)...)
	}
	if !data.RecipientDenylist.IsNull() && !data.RecipientDenylist.IsUnknown() {
		diags.Append(data.RecipientDenylist.ElementsAs(ctx, &f.RecipientDenylist, false)...)
	}
	if !data.CatchallDestinations.IsNull() && !data.CatchallDestinations.IsUnknown() {
		diags.Append(data.CatchallDestinations.ElementsAs(ctx, &f.CatchallDestinations, false)...)
	}
	if !data.CanAccess.IsNull() && !data.CanAccess.IsUnknown() {
		f.CanAccess = new(data.CanAccess.ValueBool())
	}
	if !data.MxProxyEnabled.IsNull() && !data.MxProxyEnabled.IsUnknown() {
		f.MxProxyEnabled = new(data.MxProxyEnabled.ValueBool())
	}

	return f, diags
}

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Migadu domain.\n\n~> The Migadu API does not support domain deletion. Destroying this resource removes it from Terraform state only. The domain must be deleted manually from the Migadu admin panel.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"hosted_dns": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether Migadu hosts the DNS for this domain.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"create_default_addresses": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to create default mailbox addresses on creation. Has no effect after creation.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Description of the domain.",
			},
			"tags": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Tags associated with the domain.",
			},
			"spam_aggressiveness": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Spam aggressiveness level.",
			},
			"greylisting_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether greylisting is enabled.",
			},
			"subject_rewriting_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether subject rewriting is enabled.",
			},
			"junk_subject_keyword_spam": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether junk subject keyword spam detection is enabled.",
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
			"catchall_destinations": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Catchall destination addresses.",
			},
			"can_access": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the domain can access the Migadu panel.",
			},
			"mx_proxy_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether MX proxy is enabled.",
			},
			"state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The domain state.",
			},
			"activated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the domain was activated.",
			},
			"deactivated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the domain was deactivated.",
			},
			"can_receive": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the domain can receive email.",
			},
			"can_send": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the domain can send email.",
			},
			"spf": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "SPF DNS record to add.",
				Attributes: map[string]schema.Attribute{
					"name":  schema.StringAttribute{Computed: true},
					"type":  schema.StringAttribute{Computed: true},
					"value": schema.StringAttribute{Computed: true},
				},
			},
			"dmarc": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "DMARC DNS record to add.",
				Attributes: map[string]schema.Attribute{
					"name":  schema.StringAttribute{Computed: true},
					"type":  schema.StringAttribute{Computed: true},
					"value": schema.StringAttribute{Computed: true},
				},
			},
			"dns_verification": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "DNS ownership verification record to add.",
				Attributes: map[string]schema.Attribute{
					"name":  schema.StringAttribute{Computed: true},
					"type":  schema.StringAttribute{Computed: true},
					"value": schema.StringAttribute{Computed: true},
				},
			},
			"dkim_records": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "DKIM DNS records to add (one per key).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":  schema.StringAttribute{Computed: true},
						"type":  schema.StringAttribute{Computed: true},
						"value": schema.StringAttribute{Computed: true},
					},
				},
			},
			"mx_records": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "MX DNS records to add.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":     schema.StringAttribute{Computed: true},
						"type":     schema.StringAttribute{Computed: true},
						"value":    schema.StringAttribute{Computed: true},
						"priority": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	f, diags := extractMutableFields(ctx, data.DomainModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.CreateDomainRequest{
		Name:                    data.Name.ValueString(),
		HostedDNS:               data.HostedDNS.ValueBool(),
		Description:             f.Description,
		Tags:                    f.Tags,
		SpamAggressiveness:      f.SpamAggressiveness,
		GreylistingEnabled:      f.GreylistingEnabled,
		SubjectRewritingEnabled: f.SubjectRewritingEnabled,
		JunkSubjectKeywordSpam:  f.JunkSubjectKeywordSpam,
		SenderDenylist:          f.SenderDenylist,
		SenderAllowlist:         f.SenderAllowlist,
		RecipientDenylist:       f.RecipientDenylist,
		CatchallDestinations:    f.CatchallDestinations,
		CanAccess:               f.CanAccess,
		MxProxyEnabled:          f.MxProxyEnabled,
	}
	if !data.CreateDefaultAddresses.IsNull() && !data.CreateDefaultAddresses.IsUnknown() {
		request.CreateDefaultAddresses = data.CreateDefaultAddresses.ValueBool()
	}

	domain, err := r.client.Domains().Create(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	records, err := r.client.Domains().Domain(domain.Name).Records(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain records", err.Error())
		return
	}

	domainToModel(domain, records, &data.DomainModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.Domains().Domain(data.Name.ValueString()).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	records, err := r.client.Domains().Domain(data.Name.ValueString()).Records(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain records", err.Error())
		return
	}

	domainToModel(domain, records, &data.DomainModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	f, diags := extractMutableFields(ctx, data.DomainModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.UpdateDomainRequest{
		Description:             f.Description,
		Tags:                    f.Tags,
		SpamAggressiveness:      f.SpamAggressiveness,
		GreylistingEnabled:      f.GreylistingEnabled,
		SubjectRewritingEnabled: f.SubjectRewritingEnabled,
		JunkSubjectKeywordSpam:  f.JunkSubjectKeywordSpam,
		SenderDenylist:          f.SenderDenylist,
		SenderAllowlist:         f.SenderAllowlist,
		RecipientDenylist:       f.RecipientDenylist,
		CatchallDestinations:    f.CatchallDestinations,
		CanAccess:               f.CanAccess,
		MxProxyEnabled:          f.MxProxyEnabled,
	}

	domain, err := r.client.Domains().Domain(data.Name.ValueString()).Update(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	records, err := r.client.Domains().Domain(data.Name.ValueString()).Records(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain records", err.Error())
		return
	}

	domainToModel(domain, records, &data.DomainModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.AddWarning(
		"Domain not deleted",
		fmt.Sprintf("The Migadu API does not support domain deletion. Domain %q has been removed from Terraform state but still exists in Migadu and must be removed manually.", data.Name.ValueString()),
	)
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func domainToModel(d migadu.Domain, records migadu.DomainRecords, m *DomainModel) {
	m.Name = types.StringValue(d.Name)
	m.HostedDNS = types.BoolValue(d.HostedDns)
	m.Description = types.StringValue(d.Description)
	m.SpamAggressiveness = types.StringValue(d.SpamAggressiveness)
	m.GreylistingEnabled = types.BoolValue(d.GreylistingEnabled)
	m.SubjectRewritingEnabled = types.BoolValue(d.SubjectRewritingEnabled)
	m.JunkSubjectKeywordSpam = types.BoolValue(d.JunkSubjectKeywordSpam)
	m.CanAccess = types.BoolValue(d.CanAccess)
	m.MxProxyEnabled = types.BoolValue(d.MxProxyEnabled)
	m.State = types.StringValue(d.State)
	m.ActivatedAt = types.StringValue(d.ActivatedAt)
	m.CanReceive = types.BoolValue(d.CanReceive)
	m.CanSend = types.BoolValue(d.CanSend)

	if d.DeactivatedAt != nil {
		m.DeactivatedAt = types.StringValue(*d.DeactivatedAt)
	} else {
		m.DeactivatedAt = types.StringNull()
	}

	m.Tags = stringsToList(d.Tags)
	m.SenderDenylist = stringsToList(d.SenderDenylist)
	m.SenderAllowlist = stringsToList(d.SenderAllowlist)
	m.RecipientDenylist = stringsToList(d.RecipientDenylist)
	m.CatchallDestinations = stringsToList(d.CatchallDestinations)

	m.SPF, _ = types.ObjectValue(dnsRecordAttrTypes, map[string]attr.Value{
		"name":  types.StringValue(records.SPF.Name),
		"type":  types.StringValue(records.SPF.Type),
		"value": types.StringValue(records.SPF.Value),
	})
	m.DMARC, _ = types.ObjectValue(dnsRecordAttrTypes, map[string]attr.Value{
		"name":  types.StringValue(records.DMARC.Name),
		"type":  types.StringValue(records.DMARC.Type),
		"value": types.StringValue(records.DMARC.Value),
	})
	m.DNSVerification, _ = types.ObjectValue(dnsRecordAttrTypes, map[string]attr.Value{
		"name":  types.StringValue(records.DNSVerification.Name),
		"type":  types.StringValue(records.DNSVerification.Type),
		"value": types.StringValue(records.DNSVerification.Value),
	})

	dkimElems := make([]attr.Value, len(records.DKIM))
	for i, r := range records.DKIM {
		dkimElems[i], _ = types.ObjectValue(dnsRecordAttrTypes, map[string]attr.Value{
			"name":  types.StringValue(r.Name),
			"type":  types.StringValue(r.Type),
			"value": types.StringValue(r.Value),
		})
	}
	m.DKIMRecords, _ = types.ListValue(types.ObjectType{AttrTypes: dnsRecordAttrTypes}, dkimElems)

	mxElems := make([]attr.Value, len(records.MXRecords))
	for i, r := range records.MXRecords {
		mxElems[i], _ = types.ObjectValue(mxRecordAttrTypes, map[string]attr.Value{
			"name":     types.StringValue(r.Name),
			"type":     types.StringValue(r.Type),
			"value":    types.StringValue(r.Value),
			"priority": types.Int64Value(int64(r.Priority)),
		})
	}
	m.MXRecords, _ = types.ListValue(types.ObjectType{AttrTypes: mxRecordAttrTypes}, mxElems)
}

func isNotFound(err error) bool {
	var e *migadu.Error
	if errors.As(err, &e) {
		return e.StatusCode == 404
	}
	return false
}

func stringsToList(s []string) types.List {
	elems := make([]attr.Value, len(s))
	for i, v := range s {
		elems[i] = types.StringValue(v)
	}
	list, _ := types.ListValue(types.StringType, elems)
	return list
}
