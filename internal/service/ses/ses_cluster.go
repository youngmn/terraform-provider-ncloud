package ses

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vses2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/common"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/conn"
)

const (
	SESStatusCreatingCode = "creating"
	SESStatusChangingCode = "changing"
	SESStatusWorkingCode  = "working"
	SESStatusRunningCode  = "running"
	SESStatusDeletingCode = "deleting"
	SESStatusReturnCode   = "return"
	SESStatusNullCode     = "null"
)

func ResourceNcloudSESCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNcloudSESClusterCreate,
		ReadContext:   resourceNcloudSESClusterRead,
		UpdateContext: resourceNcloudSESClusterUpdate,
		DeleteContext: resourceNcloudSESClusterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(conn.DefaultCreateTimeout),
			Update: schema.DefaultTimeout(conn.DefaultCreateTimeout),
			Delete: schema.DefaultTimeout(conn.DefaultCreateTimeout),
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"service_group_instance_no": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.All(
					validation.StringLenBetween(3, 15),
					validation.StringMatch(regexp.MustCompile(`^[a-z]+[a-z0-9]*(-[a-z0-9]+)*[a-z0-9]+$`), "Composed of alphabets(lower-case), numbers, non-consecutive hyphen (-)."),
				)),
			},
			"search_engine": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"version_code": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"port": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"dashboard_port": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"user_name": {
							Type:     schema.TypeString,
							Required: true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.All(
								validation.StringLenBetween(3, 15),
								validation.StringMatch(regexp.MustCompile(`^[a-z]+[a-z0-9-]+[a-z0-9]$`), "Allows only lowercase letters(a-z), numbers, hyphen (-). Must start with an alphabetic character, must end with an English letter or number"),
							)),
						},
						"user_password": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.All(
								validation.StringLenBetween(8, 20),
								validation.StringMatch(regexp.MustCompile(`[a-zA-Z]+`), "Must have at least one alphabet"),
								validation.StringMatch(regexp.MustCompile(`\d+`), "Must have at least one number"),
								validation.StringMatch(regexp.MustCompile(`[~!@#$%^*()\-_=\[\]\{\};:,.<>?]+`), "Must have at least one special character"),
								validation.StringMatch(regexp.MustCompile(`^[^&+\\"'/\s`+"`"+`]*$`), "Must not have ` & + \\ \" ' / and white space."),
							)),
						},
					},
				},
			},
			"os_image_code": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpc_no": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"manager_node": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"is_dual_manager": {
							Type:     schema.TypeBool,
							Required: true,
							ForceNew: true,
						},
						"subnet_no": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
						"product_code": {
							Type:     schema.TypeString,
							Required: true,
						},
						"count": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"acg_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"acg_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"data_node": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_no": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
						"product_code": {
							Type:     schema.TypeString,
							Required: true,
						},
						"count": {
							Type:             schema.TypeInt,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(3)),
						},
						"acg_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"acg_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"storage_size": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.All(
								validation.IntBetween(100, 2000),
								validation.IntDivisibleBy(10)),
							),
						},
					},
				},
			},
			"master_node": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_no": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"product_code": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"count": {
							Type:             schema.TypeInt,
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntInSlice([]int{3, 5})),
						},
						"acg_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"acg_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"manager_node_instance_no_list": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
			"cluster_node_list": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"compute_instance_no": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"compute_instance_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"private_ip": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"server_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"node_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"subnet": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"login_key_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNcloudSESClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	searchEngineParamsMap := d.Get("search_engine").([]interface{})[0].(map[string]interface{})
	dataNodeParamsMap := d.Get("data_node").([]interface{})[0].(map[string]interface{})
	managerNodeParamsMap := d.Get("manager_node").([]interface{})[0].(map[string]interface{})

	isMasterOnlyNodeActivated := false
	var masterNodeProductCode *string
	var masterNodeSubnetNo *int32
	var masterNodeCount *int32

	masterNodeParams := d.Get("master_node")
	if masterNodeParams != nil && len(masterNodeParams.([]interface{})) > 0 {
		masterNodeParamsMap := masterNodeParams.([]interface{})[0].(map[string]interface{})
		isMasterOnlyNodeActivated = true
		masterNodeProductCode = StringPtrOrNil(masterNodeParamsMap["product_code"], true)
		masterNodeSubnetNo = Int32PtrOrNil(masterNodeParamsMap["subnet_no"], true)
		masterNodeCount = Int32PtrOrNil(masterNodeParamsMap["count"], true)
	}

	var reqParams = &vses2.CreateClusterRequestVo{
		ClusterName:               StringPtrOrNil(d.GetOk("cluster_name")),
		SearchEngineVersionCode:   StringPtrOrNil(searchEngineParamsMap["version_code"], true),
		SearchEngineUserName:      StringPtrOrNil(searchEngineParamsMap["user_name"], true),
		SearchEngineUserPassword:  StringPtrOrNil(searchEngineParamsMap["user_password"], true),
		SearchEngineDashboardPort: StringPtrOrNil(searchEngineParamsMap["dashboard_port"], true),
		SoftwareProductCode:       StringPtrOrNil(d.GetOk("os_image_code")),
		VpcNo:                     Int32PtrOrNil(d.GetOk("vpc_no")),
		IsDualManager:             BoolPtrOrNil(managerNodeParamsMap["is_dual_manager"], true),
		ManagerNodeProductCode:    StringPtrOrNil(managerNodeParamsMap["product_code"], true),
		ManagerNodeSubnetNo:       Int32PtrOrNil(managerNodeParamsMap["subnet_no"], true),
		DataNodeProductCode:       StringPtrOrNil(dataNodeParamsMap["product_code"], true),
		DataNodeSubnetNo:          Int32PtrOrNil(dataNodeParamsMap["subnet_no"], true),
		DataNodeCount:             Int32PtrOrNil(dataNodeParamsMap["count"], true),
		DataNodeStorageSize:       Int32PtrOrNil(dataNodeParamsMap["storage_size"], true),
		IsMasterOnlyNodeActivated: BoolPtrOrNil(isMasterOnlyNodeActivated, true),
		MasterNodeProductCode:     masterNodeProductCode,
		MasterNodeSubnetNo:        masterNodeSubnetNo,
		MasterNodeCount:           masterNodeCount,
		LoginKeyName:              StringPtrOrNil(d.GetOk("login_key_name")),
	}

	resp, _, err := config.Client.Vses.V2Api.CreateClusterUsingPOST(ctx, *reqParams)
	if err != nil {
		LogErrorResponse("resourceNcloudSESClusterCreate", err, reqParams)
		return diag.FromErr(err)
	}
	id := strconv.Itoa(int(ncloud.Int32Value(resp.Result.ServiceGroupInstanceNo)))

	LogResponse("resourceNcloudSESClusterCreate", resp)
	if err := waitForSESClusterActive(ctx, d, config, id); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return resourceNcloudSESClusterRead(ctx, d, meta)
}

func resourceNcloudSESClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	cluster, err := GetSESCluster(ctx, config, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if cluster == nil || *cluster.ClusterStatus == SESStatusReturnCode {
		d.SetId("")
		return nil
	}

	d.SetId(ncloud.StringValue(cluster.ServiceGroupInstanceNo))
	d.Set("id", cluster.ServiceGroupInstanceNo)
	d.Set("service_group_instance_no", cluster.ServiceGroupInstanceNo)
	d.Set("cluster_name", cluster.ClusterName)
	d.Set("os_image_code", cluster.SoftwareProductCode)
	d.Set("vpc_no", cluster.VpcNo)
	d.Set("login_key_name", cluster.LoginKeyName)
	d.Set("manager_node_instance_no_list", cluster.ManagerNodeInstanceNoList)

	var userPassword string                               // API response not support user_password. Not currently available during import
	if searchEngine, ok := d.GetOk("search_engine"); ok { // Create exist in config
		searchEngineMap := searchEngine.([]interface{})[0].(map[string]interface{})
		userPassword = searchEngineMap["user_password"].(string)
	}
	searchEngineSet := schema.NewSet(schema.HashResource(ResourceNcloudSESCluster().Schema["search_engine"].Elem.(*schema.Resource)), []interface{}{})

	searchEngineSet.Add(map[string]interface{}{
		"version_code":   *cluster.SearchEngineVersionCode,
		"user_name":      *cluster.SearchEngineUserName,
		"user_password":  userPassword,
		"port":           *cluster.SearchEnginePort,
		"dashboard_port": *cluster.SearchEngineDashboardPort,
	})

	if err := d.Set("search_engine", searchEngineSet.List()); err != nil {
		log.Printf("[WARN] Error setting search_engine set for (%s): %s", d.Id(), err)
	}

	managerNodeSet := schema.NewSet(schema.HashResource(ResourceNcloudSESCluster().Schema["manager_node"].Elem.(*schema.Resource)), []interface{}{})
	managerNodeSet.Add(map[string]interface{}{
		"is_dual_manager": *cluster.IsDualManager,
		"count":           *cluster.ManagerNodeCount,
		"subnet_no":       *cluster.ManagerNodeSubnetNo,
		"product_code":    *cluster.ManagerNodeProductCode,
		"acg_id":          *cluster.ManagerNodeAcgId,
		"acg_name":        *cluster.ManagerNodeAcgName,
	})
	if err := d.Set("manager_node", managerNodeSet.List()); err != nil {
		log.Printf("[WARN] Error setting manager_node set for (%s): %s", d.Id(), err)
	}

	dataNodeSet := schema.NewSet(schema.HashResource(ResourceNcloudSESCluster().Schema["data_node"].Elem.(*schema.Resource)), []interface{}{})
	storageSize, _ := strconv.Atoi(*cluster.DataNodeStorageSize)
	dataNodeSet.Add(map[string]interface{}{
		"count":        *cluster.DataNodeCount,
		"subnet_no":    *cluster.DataNodeSubnetNo,
		"product_code": *cluster.DataNodeProductCode,
		"acg_id":       *cluster.DataNodeAcgId,
		"acg_name":     *cluster.DataNodeAcgName,
		"storage_size": storageSize,
	})
	if err := d.Set("data_node", dataNodeSet.List()); err != nil {
		log.Printf("[WARN] Error setting data_node set for (%s): %s", d.Id(), err)
	}

	if cluster.IsMasterOnlyNodeActivated != nil && *cluster.IsMasterOnlyNodeActivated {
		masterNodeSet := schema.NewSet(schema.HashResource(ResourceNcloudSESCluster().Schema["master_node"].Elem.(*schema.Resource)), []interface{}{})
		masterNodeSet.Add(map[string]interface{}{
			"count":        *cluster.MasterNodeCount,
			"subnet_no":    *cluster.MasterNodeSubnetNo,
			"product_code": *cluster.MasterNodeProductCode,
			"acg_id":       *cluster.MasterNodeAcgId,
			"acg_name":     *cluster.MasterNodeAcgName,
		})

		if err := d.Set("master_node", masterNodeSet.List()); err != nil {
			log.Printf("[WARN] Error setting master_node set for (%s): %s", d.Id(), err)
		}
	}

	clusterNodeList := schema.NewSet(schema.HashResource(ResourceNcloudSESCluster().Schema["cluster_node_list"].Elem.(*schema.Resource)), []interface{}{})
	if cluster.ClusterNodeList != nil {
		for _, clusterNode := range cluster.ClusterNodeList {
			clusterNodeList.Add(map[string]interface{}{
				"compute_instance_no":   clusterNode.ComputeInstanceNo,
				"compute_instance_name": clusterNode.ComputeInstanceName,
				"private_ip":            clusterNode.PrivateIp,
				"server_status":         clusterNode.ServerStatus,
				"node_type":             clusterNode.NodeType,
				"subnet":                clusterNode.Subnet,
			})
		}
	}
	if err := d.Set("cluster_node_list", clusterNodeList.List()); err != nil {
		log.Printf("[WARN] Error setting cluster node list for (%s): %s", d.Id(), err)
	}
	return nil
}

func resourceNcloudSESClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	if err := checkSearchEngineChanged(ctx, d, config); err != nil {
		return diag.FromErr(err)
	}
	if err := checkDataNodeChanged(ctx, d, config); err != nil {
		return diag.FromErr(err)
	}
	if err := checkNodeProductCodeChanged(ctx, d, config); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func checkSearchEngineChanged(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) error {
	if d.HasChanges("search_engine") {
		o, n := d.GetChange("search_engine")

		oldSearchEngineMap := o.([]interface{})[0].(map[string]interface{})
		newSearchEngineMap := n.([]interface{})[0].(map[string]interface{})
		if oldSearchEngineMap["user_password"] != newSearchEngineMap["user_password"] {
			LogCommonRequest("resourceNcloudSESClusterUpdate", d.Id())
			if err := waitForSESClusterActive(ctx, d, config, d.Id()); err != nil {
				return fmt.Errorf("error waiting for SES Cluster (%s) to become activating: %s", d.Id(), err)
			}

			reqParams := &vses2.ResetSearchEngineUserPasswordRequestVo{
				SearchEngineUserPassword: StringPtrOrNil(newSearchEngineMap["user_password"], true),
			}

			if _, _, err := config.Client.Vses.V2Api.ResetSearchEngineUserPasswordUsingPOST(ctx, d.Id(), reqParams); err != nil {
				LogErrorResponse("resourceNcloudSESClusterResetSearchEngineUserPassword", err, d.Id())
				return fmt.Errorf("error Reset Search Engine User Password with Cluster (%s) : %s", d.Id(), err)
			}

			if err := waitForSESClusterActive(ctx, d, config, d.Id()); err != nil {
				return fmt.Errorf("error waiting for SES Cluster (%s) to become activating: %s", d.Id(), err)
			}
		}
	}
	return nil
}

func checkDataNodeChanged(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) error {
	if d.HasChanges("data_node") {
		o, n := d.GetChange("data_node")

		oldDataNodeMap := o.([]interface{})[0].(map[string]interface{})
		newDataNodeMap := n.([]interface{})[0].(map[string]interface{})

		oldDataNodeCount := *Int32PtrOrNil(oldDataNodeMap["count"], true)
		newDataNodeCount := *Int32PtrOrNil(newDataNodeMap["count"], true)

		if oldDataNodeCount < newDataNodeCount {
			LogCommonRequest("resourceNcloudSESClusterUpdate", d.Id())
			if err := waitForSESClusterActive(ctx, d, config, d.Id()); err != nil {
				return fmt.Errorf("error waiting for SES Cluster (%s) to become activating: %s", d.Id(), err)
			}

			reqParams := &vses2.AddNodesInClusterRequestVo{
				NewDataNodeCount: StringPtrOrNil(strconv.Itoa(int(newDataNodeCount-oldDataNodeCount)), true),
			}

			if _, _, err := config.Client.Vses.V2Api.AddNodesInClusterUsingPOST(ctx, d.Id(), reqParams); err != nil {
				LogErrorResponse("resourceNcloudSESClusterAddNodes", err, d.Id())
				return fmt.Errorf("error Add Nodes to SES Cluster (%s) : %s", d.Id(), err)
			}

			if err := waitForSESClusterActive(ctx, d, config, d.Id()); err != nil {
				return fmt.Errorf("error waiting for SES Cluster (%s) to become activating: %s", d.Id(), err)
			}
		} else if oldDataNodeCount > newDataNodeCount {
			LogErrorResponse("resourceNcloudSESClusterAddNodes", nil, d.Id())
			return fmt.Errorf("data node count cannot be decreased")
		}
	}
	return nil
}

func checkNodeProductCodeChanged(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) error {
	managerNodeProductCode := getChangedNodeProductCode("manager_node", d)
	dataNodeProductCode := getChangedNodeProductCode("data_node", d)
	masterNodeProductCode := getChangedNodeProductCode("master_node", d)

	if managerNodeProductCode != nil || dataNodeProductCode != nil || masterNodeProductCode != nil {
		reqParams := &vses2.ChangeSpecNodeRequestVo{
			ManagerNodeProductCode: managerNodeProductCode,
			DataNodeProductCode:    dataNodeProductCode,
			MasterNodeProductCode:  masterNodeProductCode,
		}

		if _, _, err := config.Client.Vses.V2Api.ChangeSpecNodeUsingPOST1(ctx, d.Id(), reqParams); err != nil {
			LogErrorResponse("resourceNcloudSESClusterChangeSpec", nil, d.Id())
			return fmt.Errorf("error Change Node Product Code (%s) : %s", d.Id(), err)
		}

		if err := waitForSESClusterActive(ctx, d, config, d.Id()); err != nil {
			return fmt.Errorf("error waiting for SES Cluster (%s) to become activating: %s", d.Id(), err)
		}
	}
	return nil
}

func getChangedNodeProductCode(nodeType string, d *schema.ResourceData) *string {
	nodeParams := d.Get(nodeType)
	if nodeParams != nil && len(nodeParams.([]interface{})) > 0 {
		if d.HasChanges(nodeType) {
			o, n := d.GetChange(nodeType)
			oldNodeMap := o.([]interface{})[0].(map[string]interface{})
			newNodeMap := n.([]interface{})[0].(map[string]interface{})

			if oldNodeMap["product_code"] != newNodeMap["product_code"] {
				return StringPtrOrNil(newNodeMap["product_code"], true)
			}
		}
	}
	return nil
}

func resourceNcloudSESClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	if err := waitForSESClusterActive(ctx, d, config, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	LogCommonRequest("resourceNcloudSESClusterDelete", d.Id())
	if _, _, err := config.Client.Vses.V2Api.DeleteClusterUsingDELETE(ctx, d.Id()); err != nil {
		LogErrorResponse("resourceNcloudSESClusterDelete", err, d.Id())
		return diag.FromErr(err)
	}

	if err := waitForSESClusterDeletion(ctx, d, config); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func waitForSESClusterDeletion(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{SESStatusRunningCode, SESStatusDeletingCode},
		Target:  []string{SESStatusReturnCode},
		Refresh: func() (result interface{}, state string, err error) {
			cluster, err := GetSESCluster(ctx, config, d.Id())
			if err != nil {
				return nil, "", err
			}
			if cluster == nil {
				return d.Id(), SESStatusNullCode, nil
			}
			return cluster, ncloud.StringValue(cluster.ClusterStatus), nil
		},
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 3 * time.Second,
		Delay:      2 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("Error waiting for SES Cluster (%s) to become terminating: %s", d.Id(), err)
	}
	return nil
}

func waitForSESClusterActive(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig, id string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{SESStatusCreatingCode, SESStatusChangingCode},
		Target:  []string{SESStatusRunningCode},
		Refresh: func() (result interface{}, state string, err error) {
			cluster, err := GetSESCluster(ctx, config, id)
			if err != nil {
				return nil, "", err
			}
			if cluster == nil {
				return id, SESStatusNullCode, nil
			}
			return cluster, ncloud.StringValue(cluster.ClusterStatus), nil

		},
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 3 * time.Second,
		Delay:      2 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("error waiting for SES Cluster (%s) to become activating: %s", id, err)
	}
	return nil
}

func GetSESCluster(ctx context.Context, config *conn.ProviderConfig, id string) (*vses2.OpenApiGetClusterInfoResponseVo, error) {

	resp, _, err := config.Client.Vses.V2Api.GetClusterInfoUsingGET(ctx, id)
	if err != nil {
		return nil, err
	}
	LogResponse("GetSESCluster", resp)

	return resp.Result, nil
}

func getSESClusters(ctx context.Context, config *conn.ProviderConfig) (*vses2.GetSearchEngineClusterInfoListResponse, error) {

	resp, _, err := config.Client.Vses.V2Api.GetClusterInfoListUsingGET(ctx, nil)
	if err != nil {
		return nil, err
	}
	LogResponse("GetSESClusterList", resp)

	return resp.Result, nil
}
