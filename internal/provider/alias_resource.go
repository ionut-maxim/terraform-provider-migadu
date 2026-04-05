package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ionut-maxim/migadu"
)

var _ resource.Resource = &AliasResource{}
var _ resource.ResourceWithImportState = &AliasResource{}

type AliasResource struct {
	client *migadu.Client
}

// AliasModel is shared between resource and data source.
type AliasModel struct {
	DomainName   types.String `tfsdk:"domain_name"`
	LocalPart    types.String `tfsdk:"local_part"`
	Address      types.String `tfsdk:"address"`
	IsInternal   types.Bool   `tfsdk:"is_internal"`
	Destinations types.List   `tfsdk:"destinations"`
}

func NewAliasResource() resource.Resource {
	return &AliasResource{}
}

func (r *AliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alias"
}

func (r *AliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Migadu alias.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain this alias belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"local_part": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The local part of the alias address.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full alias address.",
			},
			"is_internal": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the alias is internal.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"destinations": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of destination addresses.",
			},
		},
	}
}

func (r *AliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.CreateAliasRequest{
		LocalPart: data.LocalPart.ValueString(),
	}
	resp.Diagnostics.Append(data.Destinations.ElementsAs(ctx, &request.Destinations, false)...)
	if !data.IsInternal.IsNull() && !data.IsInternal.IsUnknown() {
		request.IsInternal = new(data.IsInternal.ValueBool())
	}
	if resp.Diagnostics.HasError() {
		return
	}

	alias, err := r.client.Domains().Domain(data.DomainName.ValueString()).Aliases().Create(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating alias", err.Error())
		return
	}

	aliasToModel(alias, data.DomainName.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alias, err := r.client.Domains().Domain(data.DomainName.ValueString()).Aliases().Alias(data.LocalPart.ValueString()).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading alias", err.Error())
		return
	}

	aliasToModel(alias, data.DomainName.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := migadu.UpdateAliasRequest{}
	resp.Diagnostics.Append(data.Destinations.ElementsAs(ctx, &request.Destinations, false)...)
	if !data.IsInternal.IsNull() && !data.IsInternal.IsUnknown() {
		request.IsInternal = new(data.IsInternal.ValueBool())
	}
	if resp.Diagnostics.HasError() {
		return
	}

	alias, err := r.client.Domains().Domain(data.DomainName.ValueString()).Aliases().Alias(data.LocalPart.ValueString()).Update(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error updating alias", err.Error())
		return
	}

	aliasToModel(alias, data.DomainName.ValueString(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Domains().Domain(data.DomainName.ValueString()).Aliases().Alias(data.LocalPart.ValueString()).Delete(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting alias", err.Error())
	}
}

func (r *AliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: <domain_name>/<local_part>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("local_part"), parts[1])...)
}

func aliasToModel(a migadu.Alias, domainName string, m *AliasModel) {
	m.DomainName = types.StringValue(domainName)
	m.LocalPart = types.StringValue(a.LocalPart)
	m.Address = types.StringValue(a.Address)
	m.IsInternal = types.BoolValue(a.IsInternal)
	m.Destinations = stringsToList(a.Destinations)
}
