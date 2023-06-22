package provider

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/william-powell/terraform-provider-confluence/internal/confluence"
	confluencevalidators "github.com/william-powell/terraform-provider-confluence/internal/validators"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &pageResource{}
	_ resource.ResourceWithConfigure   = &pageResource{}
	_ resource.ResourceWithImportState = &pageResource{}
)

// NewItemResource is a helper function to simplify the provider implementation.
func NewPageResource() resource.Resource {
	return &pageResource{}
}

// itemResource is the resource implementation.
type pageResource struct {
	clientConfig *confluence.Config
}

// itemResourceModel maps the resource schema data.
type pageResourceModel struct {
	Id               types.Int64  `tfsdk:"id"`
	Title            types.String `tfsdk:"title"`
	Body             types.String `tfsdk:"body"`
	ParentId         types.Int64  `tfsdk:"parent_id"`
	SpaceId          types.Int64  `tfsdk:"space_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
	VersionNumber    types.Int64  `tfsdk:"version_number"`
	VersionCreatedAt types.String `tfsdk:"version_created_at"`
}

// Configure adds the provider configured client to the resource.
func (r *pageResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*confluence.Config)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	r.clientConfig = config

}

// Metadata returns the resource type name.
func (r *pageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_page"
}

// Schema defines the schema for the resource.
func (r *pageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Confluence Page. The versions of the page will be constrained to one, eliminating the need to manage versions. Changing the parent id will delete the existing page, and create a new page. Modifications directly in the Confluence UI of content will be overwritten on next apply. Changes in location or parent through the Confluence UI will yield unreliable results.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Identifier for this page.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Description: "The title for this page.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"body": schema.StringAttribute{
				Description: "The HTML body for this page",
				Required:    true,
				Validators: []validator.String{
					confluencevalidators.IsValidConfluenceHtml(),
				},
			},
			"parent_id": schema.Int64Attribute{
				Description: "The parentId of this page.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"space_id": schema.Int64Attribute{
				Description: "The space of the page",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation date for this Confluence page.",
				Computed:    true,
			},
			"version_number": schema.Int64Attribute{
				Description: "The current version number for this Confluence page.",
				Computed:    true,
			},
			"version_created_at": schema.StringAttribute{
				Description: "The creation date for this Confluence page version.",
				Computed:    true,
			},
		},
	}
}

func (r *pageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	// If our ID was a string then we could do this
	// resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	id, err := strconv.ParseInt(req.ID, 10, 64)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing page",
			"Could not import page, unexpected error (ID should be an integer): "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// Create a new resource.
func (r *pageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create page resource")
	// Retrieve values from plan
	var plan pageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	title := plan.Title.ValueString()
	body := plan.Body.ValueString()
	parentId := plan.ParentId.ValueInt64()

	newContentDetail, err := confluence.CreateNewPage(*r.clientConfig, parentId, title, body)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Item",
			err.Error(),
		)
		return
	}

	// Map response body to model
	plan.Id = types.Int64Value(newContentDetail.Id)
	plan.Body = types.StringValue(newContentDetail.Body.Storage.Value)
	plan.ParentId = types.Int64Value(newContentDetail.ParentContentId)
	plan.Title = types.StringValue(newContentDetail.Title)
	plan.SpaceId = types.Int64Value(newContentDetail.SpaceId)
	plan.CreatedAt = types.StringValue(newContentDetail.CreatedAt.Format(time.RFC822))
	plan.VersionNumber = types.Int64Value(newContentDetail.Version.Number)
	plan.VersionCreatedAt = types.StringValue(newContentDetail.Version.CreatedAt.Format(time.RFC822))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Created page resource", map[string]any{"success": true})
}

// Read resource information.
func (r *pageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read page resource")
	// Get current state
	var state pageResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	contentDetail, err := confluence.GetContentDetailById(*r.clientConfig, state.Id.ValueInt64())

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Page",
			err.Error(),
		)
		return
	}

	// Treat HTTP 404 Not Found status as a signal to remove/recreate resource
	if contentDetail.ResponseStatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if contentDetail.ResponseStatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected HTTP error code received for page",
			contentDetail.ResponseStatus,
		)
		return
	}

	// Map response body to model
	state = pageResourceModel{
		Id:               types.Int64Value(contentDetail.Id),
		Title:            types.StringValue(contentDetail.Title),
		Body:             types.StringValue(contentDetail.Body.Storage.Value),
		ParentId:         types.Int64Value(contentDetail.ParentContentId),
		SpaceId:          types.Int64Value(contentDetail.SpaceId),
		CreatedAt:        types.StringValue(contentDetail.CreatedAt.Format(time.RFC822)),
		VersionNumber:    types.Int64Value(contentDetail.Version.Number),
		VersionCreatedAt: types.StringValue(contentDetail.Version.CreatedAt.Format(time.RFC822)),
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Finished reading page resource", map[string]any{"success": true})
}

func (r *pageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update page resource")
	// Retrieve values from plan
	var plan pageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.Id.ValueInt64()
	body := plan.Body.ValueString()

	contentDetail, err := confluence.UpdateContentById(*r.clientConfig, id, body, true)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Page",
			err.Error(),
		)
		return
	}

	if contentDetail.ResponseStatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected HTTP error code received for Page",
			contentDetail.ResponseStatus,
		)
		return
	}

	plan = pageResourceModel{
		Id:               types.Int64Value(contentDetail.Id),
		Title:            types.StringValue(contentDetail.Title),
		Body:             types.StringValue(contentDetail.Body.Storage.Value),
		ParentId:         types.Int64Value(contentDetail.ParentContentId),
		SpaceId:          types.Int64Value(contentDetail.SpaceId),
		CreatedAt:        types.StringValue(contentDetail.CreatedAt.Format(time.RFC822)),
		VersionNumber:    types.Int64Value(contentDetail.Version.Number),
		VersionCreatedAt: types.StringValue(contentDetail.Version.CreatedAt.Format(time.RFC822)),
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Updated page resource", map[string]any{"success": true})
}

func (r *pageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete page resource")
	// Retrieve values from state
	var state pageResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	//delete item
	_, err := confluence.DeleteContentById(*r.clientConfig, state.Id.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Page",
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "Deleted page resource", map[string]any{"success": true})
}
