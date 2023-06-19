package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/william-powell/terraform-provider-confluence/internal/confluence"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &pageDataSource{}
	_ datasource.DataSourceWithConfigure = &pageDataSource{}
)

// NewItemDataSource is a helper function to simplify the provider implementation.
func NewPageDataSource() datasource.DataSource {
	return &pageDataSource{}
}

// pageDataSource is the data source implementation.
type pageDataSource struct {
	clientConfig *confluence.Config
}

// itemDataSourceModel maps the data source schema data.
type pageDataSourceModel struct {
	Id               types.Int64  `tfsdk:"id"`
	Title            types.String `tfsdk:"title"`
	CreatedAt        types.String `tfsdk:"created_at"`
	VersionNumber    types.Int64  `tfsdk:"version_number"`
	VersionCreatedAt types.String `tfsdk:"version_created_at"`
	SpaceId          types.Int64  `tfsdk:"space_id"`
	Body             types.String `tfsdk:"body"`
	ParentId         types.Int64  `tfsdk:"parent_id"`
}

// Configure adds the provider configured client to the data source.
func (d *pageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*confluence.Config)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	d.clientConfig = config

}

// Metadata returns the data source type name.
func (d *pageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_page"
}

// Schema defines the schema for the data source.
func (d *pageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a page.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Identifier for this Confluence page.",
				Required:    true,
			},
			"title": schema.StringAttribute{
				Description: "The title for this Confluence page.",
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
			"space_id": schema.Int64Attribute{
				Description: "The space key for this Confluence page.",
				Computed:    true,
			},
			"body": schema.StringAttribute{
				Description: "The body of the of the confluence page.",
				Computed:    true,
			},
			"parent_id": schema.Int64Attribute{
				Description: "The space key for this Confluence page.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *pageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read page data source")
	var state pageDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	contentDetail, err := confluence.GetContentDetailById(*d.clientConfig, state.Id.ValueInt64())

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Page",
			err.Error(),
		)
		return
	}

	if contentDetail.ResponseStatusCode != 200 {
		resp.Diagnostics.AddError(
			"Unable to Read Page",
			fmt.Sprintf("Status Code: %d", contentDetail.ResponseStatusCode),
		)
		return
	}

	// Map response body to model
	state = pageDataSourceModel{
		Id:               types.Int64Value(contentDetail.Id),
		Title:            types.StringValue(contentDetail.Title),
		CreatedAt:        types.StringValue(contentDetail.CreatedAt.Format(time.RFC822)),
		VersionNumber:    types.Int64Value(contentDetail.Version.Number),
		VersionCreatedAt: types.StringValue(contentDetail.Version.CreatedAt.Format(time.RFC822)),
		SpaceId:          types.Int64Value(contentDetail.SpaceId),
		Body:             types.StringValue(contentDetail.Body.Storage.Value),
		ParentId:         types.Int64Value(contentDetail.ParentContentId),
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading page data source", map[string]any{"success": true})
}
