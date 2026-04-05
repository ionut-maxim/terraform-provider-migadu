package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ datasource.DataSource = &DomainDataSource{}

type DomainDataSource struct {
	client *migadu.Client
}

// DomainDataSourceModel reuses DomainModel — all fields are identical to the resource minus create_default_addresses.
type DomainDataSourceModel = DomainModel

func NewDomainDataSource() datasource.DataSource {
	return &DomainDataSource{}
}

func (d *DomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a Migadu domain.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain name.",
			},
			"state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The domain state.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the domain.",
			},
			"tags": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Tags associated with the domain.",
			},
			"activated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the domain was activated.",
			},
			"deactivated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the domain was deactivated.",
			},
			"mx_proxy_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether MX proxy is enabled.",
			},
			"can_access": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the domain can access the Migadu panel.",
			},
			"can_receive": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the domain can receive email.",
			},
			"can_send": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the domain can send email.",
			},
			"catchall_destinations": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Catchall destination addresses.",
			},
			"greylisting_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether greylisting is enabled.",
			},
			"junk_subject_keyword_spam": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether junk subject keyword spam detection is enabled.",
			},
			"subject_rewriting_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether subject rewriting is enabled.",
			},
			"hosted_dns": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether Migadu hosts the DNS for this domain.",
			},
			"recipient_denylist": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of denied recipients.",
			},
			"sender_allowlist": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of allowed senders.",
			},
			"sender_denylist": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of denied senders.",
			},
			"spam_aggressiveness": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Spam aggressiveness level.",
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

func (d *DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DomainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := d.client.Domains().Domain(data.Name.ValueString()).Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	records, err := d.client.Domains().Domain(data.Name.ValueString()).Records(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain records", err.Error())
		return
	}

	domainToModel(domain, records, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
