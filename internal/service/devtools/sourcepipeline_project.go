package devtools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/sourcebuild"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vsourcedeploy"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vsourcepipeline"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/common"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/conn"
	. "github.com/terraform-providers/terraform-provider-ncloud/internal/verify"
)

func ResourceNcloudSourcePipeline() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNcloudSourcePipelineCreate,
		ReadContext:   resourceNcloudSourcePipelineRead,
		UpdateContext: resourceNcloudSourcePipelineUpdate,
		DeleteContext: resourceNcloudSourcePipelineDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(conn.DefaultCreateTimeout),
			Update: schema.DefaultTimeout(conn.DefaultCreateTimeout),
			Delete: schema.DefaultTimeout(conn.DefaultCreateTimeout),
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.All(
					validation.StringLenBetween(1, 30),
					validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9_-]+$`), "Composed of alphabets, numbers, hyphen (-) and underbar (_)"),
				)),
			},
			"description": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 500)),
			},
			"task": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.All(
								validation.StringLenBetween(1, 50),
								validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9_-]+$`), "Composed of alphabets, numbers, hyphen (-) and underbar (_)"),
							)),
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
								"SourceBuild", "SourceDeploy",
							}, false)),
						},
						"config": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"project_id": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"stage_id": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"scenario_id": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"target": {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"repository_name": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"repository_branch": {
													Type:     schema.TypeString,
													Optional: true,
													Computed: true,
												},
												"project_name": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"file": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"manifest": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"full_manifest": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
						"linked_tasks": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
							},
						},
					},
				},
			},
			"triggers": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"repository": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"branch": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"schedule": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"day": {
										Type:     schema.TypeList,
										Required: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"time": {
										Type:     schema.TypeString,
										Required: true,
									},
									"timezone": {
										Type:     schema.TypeString,
										Required: true,
									},
									"execute_only_with_change": {
										Type:     schema.TypeBool,
										Required: true,
									},
								},
							},
						},
						"sourcepipeline": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"name": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceNcloudSourcePipelineCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	id, err := createPipelineProject(d, config)
	if err != nil {
		return err
	}

	d.SetId(*ncloud.Int32String(ncloud.Int32Value(id)))

	return resourceNcloudSourcePipelineRead(ctx, d, meta)
}

func resourceNcloudSourcePipelineRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	pipelineProject, err := GetPipelineProject(ctx, config, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if pipelineProject == nil {
		d.SetId("")
		return nil
	}
	tasks, diags := makeTaskData(config, pipelineProject.Task)
	if diags.HasError() {
		return diags
	}

	d.SetId(*ncloud.Int32String(ncloud.Int32Value(pipelineProject.Id)))
	d.Set("name", pipelineProject.Name)
	d.Set("description", pipelineProject.Description)
	d.Set("task", tasks)
	d.Set("triggers", makeTriggerData(pipelineProject.Triggers))

	return diags
}

func resourceNcloudSourcePipelineUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	if d.HasChangesExcept("name") {
		err := updatePipelineProject(ctx, d, config, d.Id())
		if err != nil {
			return err
		}
	}

	return resourceNcloudSourcePipelineRead(ctx, d, meta)
}

func resourceNcloudSourcePipelineDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	err := deletePipelineProject(ctx, config, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func createPipelineProject(d *schema.ResourceData, config *conn.ProviderConfig) (*int32, diag.Diagnostics) {
	tasksParams, paramErr := makeVpcPipelineTaskParams(d)
	if paramErr != nil {
		return nil, paramErr
	}
	reqParams := &vsourcepipeline.CreateProject{
		Name:        ncloud.String(d.Get("name").(string)),
		Description: StringPtrOrNil(d.GetOk("description")),
		Tasks:       tasksParams,
		Trigger:     makeVpcPipelineTriggerParams(d),
	}

	LogCommonRequest("createSourcePipelineProject", reqParams)
	resp, err := config.Client.Vsourcepipeline.V1Api.CreateProject(context.Background(), reqParams)
	if err != nil {
		LogErrorResponse("createSourcePipelineProject", err, reqParams)
		return nil, diag.FromErr(err)
	}
	LogResponse("createSourcePipelineProject", resp)

	return resp.ProjectId, nil
}

func GetPipelineProject(ctx context.Context, config *conn.ProviderConfig, id string) (*PipelineProject, error) {
	LogCommonRequest("getSourcePipelineProject", id)
	resp, err := config.Client.Vsourcepipeline.V1Api.GetProject(ctx, &id)
	if err != nil {
		LogErrorResponse("getSourcePipelineProject", err, id)
		return nil, err
	}
	LogResponse("getSourcePipelineProject", resp)

	return convertVpcPipelineProject(resp), nil
}

func updatePipelineProject(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig, id string) diag.Diagnostics {
	description, ok := d.GetOk("description")
	if !ok {
		description = ""
	}
	tasksParams, paramErr := makeVpcPipelineTaskParams(d)
	if paramErr != nil {
		return paramErr
	}
	reqParams := &vsourcepipeline.ChangeProject{
		Description: ncloud.String(description.(string)),
		Tasks:       tasksParams,
		Trigger:     makeVpcPipelineTriggerParams(d),
	}

	LogCommonRequest("setSourcePipelineProject", reqParams)
	resp, err := config.Client.Vsourcepipeline.V1Api.ChangeProject(ctx, reqParams, &id)
	if err != nil {
		LogErrorResponse("setSourcePipelineProject", err, id)
		return diag.FromErr(err)
	}
	LogResponse("setSourcePipelineProject", resp)

	return nil
}

func deletePipelineProject(ctx context.Context, config *conn.ProviderConfig, id string) error {
	resp, err := config.Client.Vsourcepipeline.V1Api.DeleteProject(ctx, &id)
	if err != nil {
		LogErrorResponse("deleteSourcePipelineProject", err, id)
		return err
	}
	LogResponse("deleteSourcePipelineProject", resp)
	return nil
}

func makeVpcPipelineTaskParams(d *schema.ResourceData) ([]*vsourcepipeline.CreateProjectTasks, diag.Diagnostics) {
	var pipelineTaskParams []*vsourcepipeline.CreateProjectTasks
	taskCount := d.Get("task.#").(int)

	for i := 0; i < taskCount; i++ {
		var config *vsourcepipeline.CreateProjectConfig
		prefix := fmt.Sprintf("task.%d.", i)

		if d.Get(prefix+"type").(string) == "SourceBuild" {
			if targetBranch, ok := d.GetOk(prefix + "config.0.target.0.repository_branch"); ok {
				config = &vsourcepipeline.CreateProjectConfig{
					ProjectId: Int32PtrOrNil(d.GetOk(prefix + "config.0.project_id")),
					Target: &vsourcepipeline.CreateProjectConfigTarget{
						Info: &vsourcepipeline.CreateProjectConfigTargetInfo{
							Branch: ncloud.String(targetBranch.(string)),
						},
					},
				}
			} else {
				config = &vsourcepipeline.CreateProjectConfig{
					ProjectId: Int32PtrOrNil(d.GetOk(prefix + "config.0.project_id")),
				}
			}
		} else {
			config = &vsourcepipeline.CreateProjectConfig{
				ProjectId:  Int32PtrOrNil(d.GetOk(prefix + "config.0.project_id")),
				StageId:    Int32PtrOrNil(d.GetOk(prefix + "config.0.stage_id")),
				ScenarioId: Int32PtrOrNil(d.GetOk(prefix + "config.0.scenario_id")),
			}
		}

		err := ValidateEmptyStringElement(d.Get(prefix + "linked_tasks").([]interface{}))
		if err != nil {
			return nil, diag.Errorf("task.linkd_tasks cannot contain an empty string element")
		}
		pipelineTaskParams = append(pipelineTaskParams, &vsourcepipeline.CreateProjectTasks{
			Name:        ncloud.String(d.Get(prefix + "name").(string)),
			Type_:       ncloud.String(d.Get(prefix + "type").(string)),
			Config:      config,
			LinkedTasks: ncloud.StringInterfaceList(d.Get(prefix + "linked_tasks").([]interface{})),
		})
	}

	return pipelineTaskParams, nil
}

func makeVpcPipelineTriggerParams(d *schema.ResourceData) *vsourcepipeline.CreateProjectTrigger {
	var repositoryTrigger []*vsourcepipeline.GetRepositoryTrigger
	var scheduleTrigger []*vsourcepipeline.GetScheduleTrigger
	var sourcepipelineTrigger []*vsourcepipeline.GetPipelineTrigger
	pipelineTrigger := &vsourcepipeline.CreateProjectTrigger{}

	if _, ok := d.GetOk("triggers.0.repository"); ok {
		for _, ti := range d.Get("triggers.0.repository").(*schema.Set).List() {
			triggerInput := ti.(map[string]interface{})
			repositoryTrigger = append(repositoryTrigger, &vsourcepipeline.GetRepositoryTrigger{
				Type_:  ncloud.String(triggerInput["type"].(string)),
				Name:   ncloud.String(triggerInput["name"].(string)),
				Branch: ncloud.String(triggerInput["branch"].(string)),
			})
		}
		pipelineTrigger.Repository = repositoryTrigger
	}
	if _, ok := d.GetOk("triggers.0.schedule"); ok {
		for _, ti := range d.Get("triggers.0.schedule").(*schema.Set).List() {
			triggerInput := ti.(map[string]interface{})
			scheduleTrigger = append(scheduleTrigger, &vsourcepipeline.GetScheduleTrigger{
				Day:                    ncloud.StringInterfaceList(triggerInput["day"].([]interface{})),
				Time:                   ncloud.String(triggerInput["time"].(string)),
				TimeZone:               ncloud.String(triggerInput["timezone"].(string)),
				ScheduleOnlyWithChange: ncloud.Bool(triggerInput["execute_only_with_change"].(bool)),
			})
		}
		pipelineTrigger.Schedule = scheduleTrigger
	}
	if _, ok := d.GetOk("triggers.0.sourcepipeline"); ok {
		for _, ti := range d.Get("triggers.0.sourcepipeline").(*schema.Set).List() {
			triggerInput := ti.(map[string]interface{})
			sourcepipelineTrigger = append(sourcepipelineTrigger, &vsourcepipeline.GetPipelineTrigger{
				Id: ncloud.Int32(int32(triggerInput["id"].(int))),
			})
		}
		pipelineTrigger.SourcePipeline = sourcepipelineTrigger
	}
	return pipelineTrigger
}

func makeTaskData(config *conn.ProviderConfig, tasks []*PipelineTask) ([]map[string]interface{}, diag.Diagnostics) {
	if tasks != nil {
		var task_list []map[string]interface{}
		var diags diag.Diagnostics

		for _, task := range tasks {
			if ncloud.StringValue(task.Type_) == "SourceBuild" {
				mapping := map[string]interface{}{
					"name":         ncloud.StringValue(task.Name),
					"type":         ncloud.StringValue(task.Type_),
					"linked_tasks": ncloud.StringListValue(task.LinkedTasks),
					"config":       makeBuildTaskConfig(task.Config),
				}
				task_list = append(task_list, mapping)
				buildProject, err := getBuildProject(context.Background(), config, ncloud.Int32String(ncloud.Int32Value(task.Config.ProjectId)))
				if err != nil {
					diags = appendDiag(&diags, diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  "Invalid SourceBuild project",
						Detail:   fmt.Sprintf("Build project(project_id: %d) is not exists. Please check.", ncloud.Int32Value(task.Config.ProjectId)),
					})
				} else {
					diags = appendDiag(&diags, checkBuildTaskConfig(task.Config, buildProject.Source))
				}
			} else {
				taskConfig, err := makeDeployTaskConfig(task.Config)
				if err != nil {
					return nil, diag.FromErr(err)
				}
				mapping := map[string]interface{}{
					"name":         ncloud.StringValue(task.Name),
					"type":         ncloud.StringValue(task.Type_),
					"linked_tasks": ncloud.StringListValue(task.LinkedTasks),
					"config":       taskConfig,
				}
				task_list = append(task_list, mapping)
				deployProject, err := GetSourceDeployScenarioById(context.Background(), config, ncloud.Int32String(ncloud.Int32Value(task.Config.ProjectId)), ncloud.Int32String(ncloud.Int32Value(task.Config.StageId)), ncloud.Int32String(ncloud.Int32Value(task.Config.ScenarioId)))
				if err != nil {
					diags = appendDiag(&diags, diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  "Invalid SourceDeploy project",
						Detail:   fmt.Sprintf("Deploy project(project_id: %d, stage_id: %d, scenario_id: %d) is not exists. Please check.", ncloud.Int32Value(task.Config.ProjectId), ncloud.Int32Value(task.Config.StageId), ncloud.Int32Value(task.Config.ScenarioId)),
					})
				} else {
					diags = appendDiag(&diags, checkVpcDeployTaskConfig(task.Config, deployProject))
				}
			}
		}
		return task_list, diags
	}
	return make([]map[string]interface{}, 0), nil
}

func makeBuildTaskConfig(taskConfig *PipelineTaskConfig) []map[string]interface{} {
	if taskConfig != nil {
		target := map[string]interface{}{
			"type":              ncloud.StringValue(taskConfig.Target.Type_),
			"repository_name":   ncloud.StringValue(taskConfig.Target.Info.RepositoryName),
			"repository_branch": ncloud.StringValue(taskConfig.Target.Info.Branch),
		}
		config := map[string]interface{}{
			"project_id": ncloud.Int32Value(taskConfig.ProjectId),
			"target":     []map[string]interface{}{target},
		}
		return []map[string]interface{}{config}
	}
	return []map[string]interface{}{}
}

func checkBuildTaskConfig(taskConfig *PipelineTaskConfig, buildTarget *sourcebuild.GetProjectDetailResponseSource) diag.Diagnostic {
	if !strings.EqualFold(*taskConfig.Target.Type_, *buildTarget.Type_) {
		return diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Build target configuration have changed outside of Terraform.",
			Detail:   fmt.Sprintf("Linked repository type has changed from %s to %s. Please check.", *taskConfig.Target.Type_, *buildTarget.Type_),
		}
	} else if *buildTarget.Config.Repository != *taskConfig.Target.Info.RepositoryName {
		return diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Build target configuration have changed outside of Terraform.",
			Detail:   fmt.Sprintf("Linked repository has changed from %s to %s. Please check.", *taskConfig.Target.Info.RepositoryName, *buildTarget.Config.Repository),
		}
	}
	return diag.Diagnostic{}
}

func makeDeployTaskConfig(taskConfig *PipelineTaskConfig) ([]map[string]interface{}, error) {
	if taskConfig != nil {
		target := []map[string]interface{}{}
		if ncloud.StringValue(taskConfig.Target.Type_) == "SourceBuild" {
			deployTarget := map[string]interface{}{
				"type":         ncloud.StringValue(taskConfig.Target.Type_),
				"project_name": ncloud.StringValue(taskConfig.Target.Info.ProjectName),
			}
			target = append(target, deployTarget)
		} else if ncloud.StringValue(taskConfig.Target.Type_) == "ObjectStorage" {
			deployTarget := map[string]interface{}{
				"type": ncloud.StringValue(taskConfig.Target.Type_),
				"file": ncloud.StringValue(taskConfig.Target.Info.File),
			}
			target = append(target, deployTarget)
		} else if ncloud.StringValue(taskConfig.Target.Type_) == "KubernetesService" {
			deployTarget := map[string]interface{}{
				"type":          ncloud.StringValue(taskConfig.Target.Type_),
				"manifest":      ncloud.StringValue(taskConfig.Target.Info.Manifest),
				"full_manifest": ncloud.StringValue(taskConfig.Target.Info.FullManifest),
			}
			target = append(target, deployTarget)
		}
		config := map[string]interface{}{
			"project_id":  ncloud.Int32Value(taskConfig.ProjectId),
			"stage_id":    ncloud.Int32Value(taskConfig.StageId),
			"scenario_id": ncloud.Int32Value(taskConfig.ScenarioId),
			"target":      target,
		}
		return []map[string]interface{}{config}, nil
	}
	return nil, fmt.Errorf("Task configuration is not exists. Please check")
}

func checkVpcDeployTaskConfig(taskConfig *PipelineTaskConfig, deployTarget *vsourcedeploy.GetScenarioDetailResponse) diag.Diagnostic {
	var deployTargetType string
	if *deployTarget.Type_ == "KubernetesService" {
		deployTargetType = *deployTarget.Type_
	} else {
		deployTargetType = *deployTarget.Config.File.Type_
	}

	if *taskConfig.Target.Type_ != deployTargetType {
		return diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Deploy target configuration have changed in SourceDeploy Project.",
			Detail:   fmt.Sprintf("Target type has changed from %s to %s. Please check.", *taskConfig.Target.Type_, deployTargetType),
		}
	} else if *taskConfig.Target.Type_ == "SourceBuild" &&
		(*deployTarget.Config.File.SourceBuild.Name != *taskConfig.Target.Info.ProjectName) {
		return diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Deploy target configuration have changed in SourceDeploy Project.",
			Detail:   fmt.Sprintf("Linked repository has changed from %s to %s. Please check.", *taskConfig.Target.Info.ProjectName, *deployTarget.Config.File.SourceBuild.Name),
		}
	} else if *taskConfig.Target.Type_ == "ObjectStorage" &&
		(*deployTarget.Config.File.ObjectStorage.Bucket+"/"+*deployTarget.Config.File.ObjectStorage.Object != *taskConfig.Target.Info.File) {
		return diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Deploy target configuration have changed in SourceDeploy Project.",
			Detail:   fmt.Sprintf("Linked repository has changed from %s to %s. Please check.", *taskConfig.Target.Info.File, *deployTarget.Config.File.ObjectStorage.Bucket+"/"+*deployTarget.Config.File.ObjectStorage.Object),
		}
	} else if (*taskConfig.Target.Type_ == "KubernetesService") &&
		(strings.Join(ncloud.StringListValue(deployTarget.Config.Manifest.Path), " / ") != *taskConfig.Target.Info.FullManifest) {
		return diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Deploy target configuration have changed in SourceDeploy Project.",
			Detail:   fmt.Sprintf("Linked manifest file has changed from %s to %s. Please check.", *taskConfig.Target.Info.FullManifest, strings.Join(ncloud.StringListValue(deployTarget.Config.Manifest.Path), " / ")),
		}
	}
	return diag.Diagnostic{}
}

func makeTriggerData(triggerData *PipelineTrigger) []map[string]interface{} {
	if triggerData != nil {
		var repositoryTrigger []map[string]interface{}
		var scheduleTrigger []map[string]interface{}
		var sourcepipelineTrigger []map[string]interface{}

		for _, repo := range triggerData.Repository {
			mapping := map[string]interface{}{
				"type":   ncloud.StringValue(repo.Type_),
				"name":   ncloud.StringValue(repo.Name),
				"branch": ncloud.StringValue(repo.Branch),
			}
			repositoryTrigger = append(repositoryTrigger, mapping)
		}
		for _, schedule := range triggerData.Schedule {
			mapping := map[string]interface{}{
				"day":                      ncloud.StringListValue(schedule.Day),
				"time":                     ncloud.StringValue(schedule.Time),
				"timezone":                 ncloud.StringValue(schedule.TimeZone),
				"execute_only_with_change": ncloud.BoolValue(schedule.ExecuteOnlyWithChange),
			}
			scheduleTrigger = append(scheduleTrigger, mapping)
		}
		for _, pipeline := range triggerData.SourcePipeline {
			mapping := map[string]interface{}{
				"id":   ncloud.Int32Value(pipeline.Id),
				"name": ncloud.StringValue(pipeline.Name),
			}
			sourcepipelineTrigger = append(sourcepipelineTrigger, mapping)
		}
		triggerInfo := map[string]interface{}{
			"repository":     repositoryTrigger,
			"schedule":       scheduleTrigger,
			"sourcepipeline": sourcepipelineTrigger,
		}
		return []map[string]interface{}{triggerInfo}
	}
	return []map[string]interface{}{}
}

func appendDiag(diags *diag.Diagnostics, diag diag.Diagnostic) diag.Diagnostics {
	if diag.Summary == "" {
		return *diags
	}
	*diags = append(*diags, diag)
	return *diags
}

func convertVpcPipelineProject(r *vsourcepipeline.GetProjectDetailResponse) *PipelineProject {
	if r == nil {
		return nil
	}

	project := &PipelineProject{
		Id:          r.Id,
		Name:        r.Name,
		Description: r.Description,
	}

	for _, task := range r.Tasks {
		bitBucketWorkspace := &BitbucketWorkspace{}
		if task.Config.Target.Info.Workspace != nil {
			bitBucketWorkspace.Id = task.Config.Target.Info.Workspace.Id
			bitBucketWorkspace.Name = task.Config.Target.Info.Workspace.Name
		}

		taskTargetInfo := &PipelineTaskTargetInfo{
			RepositoryName: task.Config.Target.Info.Repository,
			Branch:         task.Config.Target.Info.Branch,
			ProjectName:    task.Config.Target.Info.ProjectName,
			File:           task.Config.Target.Info.File,
			Manifest:       task.Config.Target.Info.Manifest,
			FullManifest:   task.Config.Target.Info.FullManifest,
			Workspace:      bitBucketWorkspace,
		}

		taskTarget := &PipelineTaskTarget{
			Type_: task.Config.Target.Type_,
			Info:  taskTargetInfo,
		}

		config := &PipelineTaskConfig{
			ProjectId:  task.Config.ProjectId,
			StageId:    task.Config.StageId,
			ScenarioId: task.Config.ScenarioId,
			Target:     taskTarget,
		}

		ti := &PipelineTask{
			Id:          task.Id,
			Name:        task.Name,
			Type_:       task.Type_,
			Config:      config,
			LinkedTasks: task.LinkedTasks,
		}

		project.Task = append(project.Task, ti)
	}

	if r.Trigger != nil {
		trigger := &PipelineTrigger{}
		for _, repositoryInfo := range r.Trigger.Repository {
			ri := &PipelineTriggerRepository{
				Type_:  repositoryInfo.Type_,
				Name:   repositoryInfo.Name,
				Branch: repositoryInfo.Branch,
			}
			trigger.Repository = append(trigger.Repository, ri)
		}
		for _, scheduleInfo := range r.Trigger.Schedule {
			ri := &PipelineTriggerSchedule{
				Day:                   scheduleInfo.Day,
				Time:                  scheduleInfo.Time,
				TimeZone:              scheduleInfo.TimeZone,
				ExecuteOnlyWithChange: scheduleInfo.ScheduleOnlyWithChange,
			}
			trigger.Schedule = append(trigger.Schedule, ri)
		}
		for _, pipelineInfo := range r.Trigger.SourcePipeline {
			ri := &PipelineTriggerSourcePipeline{
				Id:   pipelineInfo.Id,
				Name: pipelineInfo.Name,
			}
			trigger.SourcePipeline = append(trigger.SourcePipeline, ri)
		}

		if len(r.Trigger.Repository) != 0 || len(r.Trigger.Schedule) != 0 || len(r.Trigger.SourcePipeline) != 0 {
			project.Triggers = trigger
		}
	}

	return project
}

type PipelineProject struct {
	Id *int32 `json:"id,omitempty"`

	Name *string `json:"name,omitempty"`

	Description *string `json:"description,omitempty"`

	Task []*PipelineTask `json:"tasks,omitempty"`

	Triggers *PipelineTrigger `json:"trigger,omitempty"`
}

type PipelineTask struct {
	Id *int32 `json:"id,omitempty"`

	Name *string `json:"name,omitempty"`

	Type_ *string `json:"type,omitempty"`

	Config *PipelineTaskConfig `json:"config,omitempty"`

	LinkedTasks []*string `json:"linkedTasks,omitempty"`
}

type PipelineTaskConfig struct {
	ProjectId *int32 `json:"projectId,omitempty"`

	StageId *int32 `json:"stageId,omitempty"`

	ScenarioId *int32 `json:"scenarioId,omitempty"`

	Target *PipelineTaskTarget `json:"target,omitempty"`
}

type PipelineTaskTarget struct {
	Type_ *string `json:"type,omitempty"`

	Info *PipelineTaskTargetInfo `json:"info,omitempty"`
}

type PipelineTaskTargetInfo struct {
	RepositoryName *string `json:"repository,omitempty"`

	Branch *string `json:"branch,omitempty"`

	Workspace *BitbucketWorkspace `json:"workspace,omitempty"`

	ProjectName *string `json:"projectName,omitempty"`

	File *string `json:"file,omitempty"`

	Manifest *string `json:"manifest,omitempty"`

	FullManifest *string `json:"fullManifest,omitempty"`
}

type BitbucketWorkspace struct {
	Id *string `json:"id,omitempty"`

	Name *string `json:"name,omitempty"`
}

type PipelineTrigger struct {
	Repository []*PipelineTriggerRepository `json:"repository,omitempty"`

	Schedule []*PipelineTriggerSchedule `json:"schedule,omitempty"`

	SourcePipeline []*PipelineTriggerSourcePipeline `json:"sourcepipeline,omitempty"`
}

type PipelineTriggerRepository struct {
	Type_ *string `json:"type,omitempty"`

	Name *string `json:"name,omitempty"`

	Branch *string `json:"branch,omitempty"`
}

type PipelineTriggerSchedule struct {
	Day []*string `json:"day,omitempty"`

	Time *string `json:"time,omitempty"`

	TimeZone *string `json:"timeZone,omitempty"`

	ExecuteOnlyWithChange *bool `json:"scheduleOnlyWithChange,omitempty"`
}

type PipelineTriggerSourcePipeline struct {
	Id *int32 `json:"id,omitempty"`

	Name *string `json:"name,omitempty"`
}
