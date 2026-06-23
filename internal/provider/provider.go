package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var _ provider.Provider = (*ScaleGridProvider)(nil)

// ScaleGridProvider is the provider implementation.
type ScaleGridProvider struct {
	version string
}

// ScaleGridProviderModel maps provider configuration to Go types.
type ScaleGridProviderModel struct {
	BaseURL       types.String `tfsdk:"base_url"`
	Email         types.String `tfsdk:"email"`
	Password      types.String `tfsdk:"password"`
	TwoFactorCode types.String `tfsdk:"two_factor_code"`
}

// New returns a constructor that captures the provider version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ScaleGridProvider{version: version}
	}
}

func (p *ScaleGridProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "scalegrid"
	resp.Version = p.version
}

func (p *ScaleGridProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The ScaleGrid provider manages database deployments (MongoDB, Redis, MySQL, " +
			"PostgreSQL) and related resources through the ScaleGrid console API.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional: true,
				Description: "Base URL of the ScaleGrid console. Defaults to `" + client.DefaultBaseURL +
					"`. May also be set with `SCALEGRID_BASE_URL`. For a dedicated/on-prem controller, " +
					"set this to your controller's URL.",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Description: "ScaleGrid account email. May also be set with `SCALEGRID_EMAIL`.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "ScaleGrid account password. May also be set with `SCALEGRID_PASSWORD`.",
			},
			"two_factor_code": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Optional two-factor (TOTP) code. Because TOTP codes expire within seconds, " +
					"this is only practical for one-shot runs; for automation, use an account with 2FA " +
					"disabled. May also be set with `SCALEGRID_TWO_FACTOR_CODE`.",
			},
		},
	}
}

func (p *ScaleGridProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ScaleGridProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := firstNonEmpty(stringValue(config.BaseURL), os.Getenv("SCALEGRID_BASE_URL"))
	email := firstNonEmpty(stringValue(config.Email), os.Getenv("SCALEGRID_EMAIL"))
	password := firstNonEmpty(stringValue(config.Password), os.Getenv("SCALEGRID_PASSWORD"))
	twoFactor := firstNonEmpty(stringValue(config.TwoFactorCode), os.Getenv("SCALEGRID_TWO_FACTOR_CODE"))

	if email == "" {
		resp.Diagnostics.AddAttributeError(path.Root("email"), "Missing ScaleGrid email",
			"Set the `email` attribute or the `SCALEGRID_EMAIL` environment variable.")
	}
	if password == "" {
		resp.Diagnostics.AddAttributeError(path.Root("password"), "Missing ScaleGrid password",
			"Set the `password` attribute or the `SCALEGRID_PASSWORD` environment variable.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.NewClient(ctx, client.Config{
		BaseURL:       baseURL,
		Email:         email,
		Password:      password,
		TwoFactorCode: twoFactor,
		UserAgent:     "terraform-provider-scalegrid/" + p.version,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to authenticate with ScaleGrid", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *ScaleGridProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterResource,
		NewCloudProfileResource,
		NewFirewallResource,
		NewAlertRuleResource,
		NewBackupResource,
		NewFollowerResource,
	}
}

func (p *ScaleGridProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewClusterDataSource,
		NewClustersDataSource,
		NewCloudProfileDataSource,
		NewDatabaseVersionsDataSource,
		NewClusterCredentialsDataSource,
	}
}
