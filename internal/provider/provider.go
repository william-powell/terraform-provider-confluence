package provider

import (
	"context"
	"os"

	"github.com/william-powell/terraform-provider-confluence/internal/confluence"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &confluenceProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &confluenceProvider{
			version: version,
		}
	}
}

// confluenceProvider is the provider implementation.
type confluenceProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// confluenceProviderModel maps provider schema data to a Go type.
type confluenceProviderModel struct {
	BaseUrl  types.String `tfsdk:"base_url"`
	Username types.String `tfsdk:"username"`
	Apikey   types.String `tfsdk:"api_key"`
}

// Metadata returns the provider type name.
func (p *confluenceProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "confluence"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *confluenceProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "The hostname confluence cloud service endpoint. May also be provided via the CONFLUENCE_BASE_URL environment variable.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "The username of the confluence cloud API credentials. May also be provided via the CONFLUENCE_USERNAME environment variable.",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Description: "The apikey of the confluence cloud API credentials. May also be provided via the CONFLUENCE_API_KEY environment variable.",
			},
		},
		Blocks:      map[string]schema.Block{},
		Description: "Interface with the Confluence Cloud service API.",
	}
}

// Configure prepares a Inventory API client for data sources and resources.
//
//gocyclo:ignore
func (p *confluenceProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Confluence client")

	// Retrieve provider data from configuration
	var config confluenceProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.BaseUrl.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown Confluence Cloud API Base Url",
			"The provider cannot create the Confluence API client as there is an unknown configuration value for the Confluence API base url. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CONFLUENCE_BASE_URL environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown Confluence Cloud API Username",
			"The provider cannot create the Confluence API client as there is an unknown configuration value for the Confluence API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CONFLUENCE_USERNAME environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Confluence Cloud API Key",
			"The provider cannot create the Confluence API client as there is an unknown configuration value for the Confluence API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CONFLUENCE_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	baseurl := os.Getenv("CONFLUENCE_BASE_URL")
	username := os.Getenv("CONFLUENCE_USERNAME")
	apikey := os.Getenv("CONFLUENCE_API_KEY")

	if !config.BaseUrl.IsNull() {
		baseurl = config.BaseUrl.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Apikey.IsNull() {
		apikey = config.Apikey.ValueString()
	}

	// // If any of the expected configurations are missing, return
	// // errors with provider-specific guidance.

	if baseurl == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Missing Confluence API Base Url",
			"The provider is missing or empty value for the Confluence API base_url. "+
				"Set the Confluence API base value (e.g. https://<unique>.atlassian.net) in the configuration or use the CONFLUENCE_BASE_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		baseurl = "unknown"
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Confluence API username",
			"The provider is missing or empty value for the Confluence API username. "+
				"Set the Confluence API user value in the configuration or use the CONFLUENCE_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		username = "unknown"
	}

	if apikey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Confluence API key",
			"The provider is missing or empty value for the Confluence API key. "+
				"Set the Confluence API key value in the configuration or use the CONFLUENCE_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		apikey = "unknown"
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Confluence client")

	confluenceApiConfig := confluence.NewConfig(baseurl, username, apikey)

	contentDetail, err := confluence.GetContentDetailById(*confluenceApiConfig, int64(1))
	_ = contentDetail

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Confluence API Client",
			"An unexpected error occurred when creating the Confluence API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Confluence Client Error: "+err.Error(),
		)
		return
	}

	// Make the Confluence client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = confluenceApiConfig
	resp.ResourceData = confluenceApiConfig

	tflog.Info(ctx, "Configured Confluence client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *confluenceProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPageDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *confluenceProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPageResource,
	}
}
