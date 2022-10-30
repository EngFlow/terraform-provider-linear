package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/terraform-community-providers/terraform-plugin-framework-utils/modifiers"
	"github.com/terraform-community-providers/terraform-plugin-framework-utils/validators"
)

var _ resource.Resource = &TeamLabelResource{}
var _ resource.ResourceWithImportState = &TeamLabelResource{}

func NewTeamLabelResource() resource.Resource {
	return &TeamLabelResource{}
}

type TeamLabelResource struct {
	client *graphql.Client
}

type TeamLabelResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Color       types.String `tfsdk:"color"`
	TeamId      types.String `tfsdk:"team_id"`
}

func (r *TeamLabelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_label"
}

func (r *TeamLabelResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Linear team label.",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "Identifier of the label.",
				Type:                types.StringType,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"name": {
				MarkdownDescription: "Name of the label.",
				Type:                types.StringType,
				Required:            true,
				Validators: []tfsdk.AttributeValidator{
					validators.MinLength(1),
				},
			},
			"description": {
				MarkdownDescription: "Description of the label.",
				Type:                types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					modifiers.NullableString(),
				},
			},
			"color": {
				MarkdownDescription: "Color of the label.",
				Type:                types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
				Validators: []tfsdk.AttributeValidator{
					validators.Match(colorRegex()),
				},
			},
			"team_id": {
				MarkdownDescription: "Identifier of the team.",
				Type:                types.StringType,
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
				Validators: []tfsdk.AttributeValidator{
					validators.Match(uuidRegex()),
				},
			},
		},
	}, nil
}

func (r *TeamLabelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *TeamLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *TeamLabelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := IssueLabelCreateInput{
		Name:   data.Name.Value,
		TeamId: &data.TeamId.Value,
	}

	if !data.Description.IsNull() {
		input.Description = &data.Description.Value
	}

	if !data.Color.IsUnknown() {
		input.Color = &data.Color.Value
	}

	response, err := createLabel(context.Background(), *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create team label, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a team label")

	issueLabel := response.IssueLabelCreate.IssueLabel

	data.Id = types.String{Value: issueLabel.Id}
	data.Name = types.String{Value: issueLabel.Name}

	if issueLabel.Description != nil {
		data.Description = types.String{Value: *issueLabel.Description}
	}

	if issueLabel.Color != nil {
		data.Color = types.String{Value: *issueLabel.Color}
	}

	if issueLabel.Team != nil {
		data.TeamId = types.String{Value: issueLabel.Team.Id}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *TeamLabelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getLabel(context.Background(), *r.client, data.Id.Value)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team label, got error: %s", err))
		return
	}

	issueLabel := response.IssueLabel

	data.Id = types.String{Value: issueLabel.Id}
	data.Name = types.String{Value: issueLabel.Name}

	if issueLabel.Description != nil {
		data.Description = types.String{Value: *issueLabel.Description}
	}

	if issueLabel.Color != nil {
		data.Color = types.String{Value: *issueLabel.Color}
	}

	if issueLabel.Team != nil {
		data.TeamId = types.String{Value: issueLabel.Team.Id}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *TeamLabelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := IssueLabelUpdateInput{
		Name: data.Name.Value,
	}

	if !data.Description.IsNull() {
		input.Description = &data.Description.Value
	}

	if !data.Color.IsUnknown() {
		input.Color = &data.Color.Value
	}

	response, err := updateLabel(context.Background(), *r.client, input, data.Id.Value)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update team label, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a team label")

	issueLabel := response.IssueLabelUpdate.IssueLabel

	data.Id = types.String{Value: issueLabel.Id}
	data.Name = types.String{Value: issueLabel.Name}

	if issueLabel.Description != nil {
		data.Description = types.String{Value: *issueLabel.Description}
	}

	if issueLabel.Color != nil {
		data.Color = types.String{Value: *issueLabel.Color}
	}

	if issueLabel.Team != nil {
		data.TeamId = types.String{Value: issueLabel.Team.Id}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *TeamLabelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteLabel(context.Background(), *r.client, data.Id.Value)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete team label, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a team label")
}

func (r *TeamLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: label_name:team_key. Got: %q", req.ID),
		)

		return
	}

	response, err := findTeamLabel(context.Background(), *r.client, parts[0], parts[1])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import team label, got error: %s", err))
		return
	}

	if len(response.IssueLabels.Nodes) != 1 {
		resp.Diagnostics.AddError("Client Error", "Unable to import team label, got error: label not found")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), response.IssueLabels.Nodes[0].Id)...)
}
