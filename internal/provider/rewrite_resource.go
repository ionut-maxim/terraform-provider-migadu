package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ resource.Resource = &RewriteResource{}
var _ resource.ResourceWithImportState = &RewriteResource{}

type RewriteResource struct {
	client *migadu.Client
}

// RewriteModel is shared between resource and data source.
type RewriteModel struct {
	DomainName    types.String `tfsdk:"domain_name"`
	Name          types.String `tfsdk:"name"`
	LocalPartRule types.String `tfsdk:"local_part_rule"`
	OrderNum      types.Int64  `tfsdk:"order_num"`
	Destinations  types.List   `tfsdk:"destinations"`
}

func NewRewriteResource() resource.Resource {
	return &RewriteResource{}
}

func (r *RewriteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rewrite"
}

func (r *RewriteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Migadu rewrite rule.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain this rewrite rule belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name (identifier) of the rewrite rule.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"local_part_rule": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part rule pattern.",
			},
			"order_num": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The ordering number of the rule.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"destinations": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of destination addresses.",
			},
		},
	}
}

func (r *RewriteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RewriteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RewriteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.CreateRewriteRequest{
		Name:          data.Name.ValueString(),
		LocalPartRule: data.LocalPartRule.ValueString(),
	}
	resp.Diagnostics.Append(data.Destinations.ElementsAs(ctx, &request.Destinations, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rewrite, err := r.client.Domains().Domain(data.DomainName.ValueString()).Rewrites().Create(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rewrite",
			err.Error()+"\n\nNote: rewrite rules require the domain to be active (DNS records verified).",
		)
		return
	}

	rewriteToModel(rewrite, data.DomainName.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RewriteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RewriteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rewrite, err := r.client.Domains().Domain(data.DomainName.ValueString()).Rewrites().Rewrite(data.Name.ValueString()).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading rewrite", err.Error())
		return
	}

	rewriteToModel(rewrite, data.DomainName.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RewriteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RewriteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.UpdateRewriteRequest{
		LocalPartRule: data.LocalPartRule.ValueString(),
	}
	resp.Diagnostics.Append(data.Destinations.ElementsAs(ctx, &request.Destinations, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rewrite, err := r.client.Domains().Domain(data.DomainName.ValueString()).Rewrites().Rewrite(data.Name.ValueString()).Update(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error updating rewrite", err.Error())
		return
	}

	rewriteToModel(rewrite, data.DomainName.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RewriteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RewriteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Domains().Domain(data.DomainName.ValueString()).Rewrites().Rewrite(data.Name.ValueString()).Delete(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting rewrite", err.Error())
	}
}

func (r *RewriteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: <domain_name>/<name>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

func rewriteToModel(r migadu.Rewrite, domainName string, m *RewriteModel) {
	m.DomainName = types.StringValue(domainName)
	m.Name = types.StringValue(r.Name)
	m.LocalPartRule = types.StringValue(r.LocalPartRule)
	m.OrderNum = types.Int64Value(int64(r.OrderNum))
	m.Destinations = stringsToList(r.Destinations)
}
