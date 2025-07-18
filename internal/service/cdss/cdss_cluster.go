package cdss

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vcdss"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/common"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/conn"
)

const (
	CDSSStatusCreating = "creating"
	CDSSStatusChanging = "changing"
	CDSSStatusRunning  = "running"
	CDSSStatusDeleting = "deleting"
	CDSSStatusError    = "error"
	CDSSStatusReturn   = "return"
	CDSSStatusNull     = "null"
)

func ResourceNcloudCDSSCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNcloudCDSSClusterCreate,
		ReadContext:   resourceNcloudCDSSClusterRead,
		UpdateContext: resourceNcloudCDSSClusterUpdate,
		DeleteContext: resourceNcloudCDSSClusterDelete,
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
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.All(
					validation.StringLenBetween(3, 15),
					validation.StringMatch(regexp.MustCompile(`^[a-z]+[a-z0-9-]+[a-z0-9]$`), "Allows only lowercase letters(a-z), numbers, hyphen (-). Must start with an alphabetic character, must end with an English letter or number"))),
			},
			"kafka_version_code": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"os_image": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpc_no": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"config_group_no": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cmak": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user_name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
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
			"manager_node": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"node_product_code": {
							Type:     schema.TypeString,
							Required: true,
						},
						"subnet_no": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"broker_nodes": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"node_product_code": {
							Type:     schema.TypeString,
							Required: true,
						},
						"subnet_no": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"node_count": {
							Type:             schema.TypeInt,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(3, 10)),
						},
						"storage_size": {
							Type:             schema.TypeInt,
							Required:         true,
							ForceNew:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(100, 2000)),
						},
					},
				},
			},
			"endpoints": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"plaintext": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"tls": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"public_endpoint_plaintext_listener_port": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"public_endpoint_tls_listener_port": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"public_endpoint_plaintext": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"public_endpoint_tls": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"zookeeper": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"hosts_private_endpoint_tls": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"hosts_public_endpoint_tls": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceNcloudCDSSClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	c := d.Get("cmak").([]interface{})
	cMap := c[0].(map[string]interface{})

	m := d.Get("manager_node").([]interface{})
	mMap := m[0].(map[string]interface{})

	b := d.Get("broker_nodes").([]interface{})
	bMap := b[0].(map[string]interface{})

	reqParams := vcdss.CreateCluster{
		ClusterName:              *StringPtrOrNil(d.GetOk("name")),
		KafkaVersionCode:         *StringPtrOrNil(d.GetOk("kafka_version_code")),
		KafkaManagerUserName:     *StringPtrOrNil(cMap["user_name"], true),
		KafkaManagerUserPassword: *StringPtrOrNil(cMap["user_password"], true),
		SoftwareProductCode:      *StringPtrOrNil(d.GetOk("os_image")),
		VpcNo:                    *GetInt32FromString(d.GetOk("vpc_no")),
		ManagerNodeProductCode:   *StringPtrOrNil(mMap["node_product_code"], true),
		ManagerNodeSubnetNo:      *GetInt32FromString(mMap["subnet_no"], true),
		BrokerNodeProductCode:    *StringPtrOrNil(bMap["node_product_code"], true),
		BrokerNodeCount:          *Int32PtrOrNil(bMap["node_count"], true),
		BrokerNodeSubnetNo:       *GetInt32FromString(bMap["subnet_no"], true),
		BrokerNodeStorageSize:    *Int32PtrOrNil(bMap["storage_size"], true),
		ConfigGroupNo:            *GetInt32FromString(d.GetOk("config_group_no")),
	}

	resp, _, err := config.Client.Vcdss.V1Api.ClusterCreateCDSSClusterReturnServiceGroupInstanceNoPost(ctx, reqParams)
	if err != nil {
		LogErrorResponse("resourceNcloudCDSSClusterCreate", err, reqParams)
		return diag.FromErr(err)
	}
	LogResponse("resourceNcloudCDSSClusterCreate", resp)

	id := strconv.Itoa(int(ncloud.Int32Value(&resp.Result.ServiceGroupInstanceNo)))
	if err := waitForCDSSClusterActive(ctx, d, config, id); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return resourceNcloudCDSSClusterRead(ctx, d, meta)
}

func resourceNcloudCDSSClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	cluster, err := getCDSSCluster(ctx, config, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if cluster == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", cluster.ClusterName)
	d.Set("kafka_version_code", cluster.KafkaVersionCode)
	d.Set("os_image", cluster.SoftwareProductCode)
	d.Set("vpc_no", strconv.Itoa(int(cluster.VpcNo)))
	d.Set("config_group_no", strconv.Itoa(int(cluster.ConfigGroupNo)))

	var cList []map[string]interface{}
	var mList []map[string]interface{}
	var bList []map[string]interface{}
	var eList []map[string]interface{}

	var userPassword string           // API response not support user_password. Not currently available during import
	if c, ok := d.GetOk("cmak"); ok { // Create exist in config
		cMap := c.([]interface{})[0].(map[string]interface{})
		userPassword = cMap["user_password"].(string)
	}

	cList = append(cList, map[string]interface{}{
		"user_name":     cluster.KafkaManagerUserName,
		"user_password": userPassword,
	})

	mList = append(mList, map[string]interface{}{
		"node_product_code": cluster.ManagerNodeProductCode,
		"subnet_no":         strconv.Itoa(int(cluster.ManagerNodeSubnetNo)),
	})
	bList = append(bList, map[string]interface{}{
		"node_product_code": cluster.BrokerNodeProductCode,
		"subnet_no":         strconv.Itoa(int(cluster.BrokerNodeSubnetNo)),
		"node_count":        cluster.BrokerNodeCount,
		"storage_size":      cluster.BrokerNodeStorageSize,
	})

	endpoints, err := getBrokerInfo(ctx, config, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	commaSplitFn := func(c rune) bool {
		return c == ','
	}
	newlineSplitFn := func(c rune) bool {
		return c == '\n'
	}
	eList = append(eList, map[string]interface{}{
		"plaintext": strings.FieldsFunc(endpoints.BrokerNodeList, commaSplitFn),
		"tls":       strings.FieldsFunc(endpoints.BrokerTlsNodeList, commaSplitFn),
		"public_endpoint_plaintext_listener_port": strings.FieldsFunc(endpoints.PublicEndpointBrokerNodeListenerPortList, newlineSplitFn),
		"public_endpoint_tls_listener_port":       strings.FieldsFunc(endpoints.PublicEndpointBrokerTlsNodeListenerPortList, newlineSplitFn),
		"public_endpoint_plaintext":               strings.FieldsFunc(endpoints.PublicEndpointBrokerNodeList, newlineSplitFn),
		"public_endpoint_tls":                     strings.FieldsFunc(endpoints.PublicEndpointBrokerTlsNodeList, newlineSplitFn),
		"zookeeper":                               strings.FieldsFunc(endpoints.ZookeeperList, commaSplitFn),
		"hosts_private_endpoint_tls":              strings.FieldsFunc(endpoints.LocalDnsList, newlineSplitFn),
		"hosts_public_endpoint_tls":               strings.FieldsFunc(endpoints.LocalDnsTlsList, newlineSplitFn),
	})

	// Only set data intersection between resource and list
	if err := d.Set("cmak", cList); err != nil {
		log.Printf("[WARN] Error setting cmak set for (%s): %s", d.Id(), err)
	}

	if err := d.Set("manager_node", mList); err != nil {
		log.Printf("[WARN] Error setting manager_node set for (%s): %s", d.Id(), err)
	}

	if err := d.Set("broker_nodes", bList); err != nil {
		log.Printf("[WARN] Error setting broker_nodes set for (%s): %s", d.Id(), err)
	}

	if err := d.Set("endpoints", eList); err != nil {
		log.Printf("[WARN] Error setting endpoints set for (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceNcloudCDSSClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	checkConfigGroupNoChanged(ctx, d, config)
	checkCmakPasswordChanged(ctx, d, config)
	if err := checkNodeCountChanged(ctx, d, config); err != nil {
		return diag.FromErr(err)
	}
	if err := checkCDSSNodeProductCodeChanged(ctx, d, config); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func checkConfigGroupNoChanged(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) diag.Diagnostics {
	if d.HasChanges("config_group_no") {
		_, n := d.GetChange("config_group_no")

		newConfigGroupNo := n.(string)
		LogCommonRequest("resourceNcloudCDSSClusterUpdate", d.Id())
		if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
			return diag.FromErr(err)
		}

		reqParams := vcdss.SetClusterKafkaConfigGroupRequest{
			KafkaVersionCode:       *StringPtrOrNil(d.GetOk("kafka_version_code")),
			ServiceGroupInstanceNo: *GetInt32FromString(d.Id(), true),
		}

		if _, _, err := config.Client.Vcdss.V1Api.ConfigGroupSetClusterKafkaConfigGroupConfigGroupNoPost(ctx, reqParams, newConfigGroupNo); err != nil {
			LogErrorResponse("resourceNcloudCDSSClusterUpdate", err, d.Id())
			return diag.FromErr(err)
		}
		if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func checkCmakPasswordChanged(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) diag.Diagnostics {
	if d.HasChanges("cmak") {
		o, n := d.GetChange("cmak")

		oldCmakMap := o.([]interface{})[0].(map[string]interface{})
		newCmakMap := n.([]interface{})[0].(map[string]interface{})
		if oldCmakMap["user_password"] != newCmakMap["user_password"] {
			LogCommonRequest("resourceNcloudCDSSClusterUpdate", d.Id())
			if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
				return diag.FromErr(err)
			}

			reqParams := vcdss.ResetCmakPassword{
				KafkaManagerUserPassword: *StringPtrOrNil(newCmakMap["user_password"], true),
			}

			if _, _, err := config.Client.Vcdss.V1Api.ClusterResetCMAKPasswordServiceGroupInstanceNoPost(ctx, reqParams, d.Id()); err != nil {
				LogErrorResponse("resourceNcloudCDSSClusterResetCmakUserPassword", err, d.Id())
				return diag.FromErr(err)
			}

			if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	return nil
}

func checkNodeCountChanged(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) error {
	if d.HasChanges("broker_nodes") {
		o, n := d.GetChange("broker_nodes")

		oldBrokerNodesMap := o.([]interface{})[0].(map[string]interface{})
		newBrokerNodesMap := n.([]interface{})[0].(map[string]interface{})

		oldDataNodeCount := *Int32PtrOrNil(oldBrokerNodesMap["node_count"], true)
		newDataNodeCount := *Int32PtrOrNil(newBrokerNodesMap["node_count"], true)

		if oldDataNodeCount < newDataNodeCount {
			LogCommonRequest("resourceNcloudCDSSClusterUpdate", d.Id())
			if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
				return fmt.Errorf("error waiting for CDSS Cluster (%s) to become activating: %s", d.Id(), err)
			}

			reqParams := vcdss.AddNodesInCluster{
				NewBrokerNodeCount: newDataNodeCount - oldDataNodeCount,
			}

			if _, _, err := config.Client.Vcdss.V1Api.ClusterChangeCountOfBrokerNodeServiceGroupInstanceNoPost(ctx, reqParams, d.Id()); err != nil {
				LogErrorResponse("resourceNcloudCDSSClusterAddNodes", err, d.Id())
				return fmt.Errorf("error Add Nodes to CDSS Cluster (%s) : %s", d.Id(), err)
			}

			if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
				return fmt.Errorf("error waiting for CDSS Cluster (%s) to become activating: %s", d.Id(), err)
			}
		} else if oldDataNodeCount > newDataNodeCount {
			LogErrorResponse("resourceNcloudCDSSClusterAddNodes", nil, d.Id())
			return fmt.Errorf("broker node count cannot be decreased")
		}
	}
	return nil
}

func checkCDSSNodeProductCodeChanged(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) error {
	managerNodeProductCode := getChangedCDSSNodeProductCode("manager_node", d)
	brokerNodeProductCode := getChangedCDSSNodeProductCode("broker_nodes", d)

	if managerNodeProductCode != nil || brokerNodeProductCode != nil {
		if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
			return fmt.Errorf("error waiting for CDSS Cluster (%s) to become activating: %s", d.Id(), err)
		}
		reqParams := vcdss.ChangeSpecNodeRequestVo{
			ManagerNodeProductCode: *managerNodeProductCode,
			BrokerNodeProductCode:  *brokerNodeProductCode,
		}

		if _, _, err := config.Client.Vcdss.V1Api.ClusterChangeSpecNodeServiceGroupInstanceNoPost(ctx, reqParams, d.Id()); err != nil {
			LogErrorResponse("resourceNcloudCDSSClusterChangeSpec", nil, d.Id())
			return fmt.Errorf("error Change Node Product Code (%s) : %s", d.Id(), err)
		}

		if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
			return fmt.Errorf("error waiting for CDSS Cluster (%s) to become activating: %s", d.Id(), err)
		}
	}
	return nil
}

func getChangedCDSSNodeProductCode(nodeType string, d *schema.ResourceData) *string {
	nodeParams := d.Get(nodeType)
	if nodeParams != nil && len(nodeParams.([]interface{})) > 0 {
		if d.HasChanges(nodeType) {
			o, n := d.GetChange(nodeType)
			oldNodeMap := o.([]interface{})[0].(map[string]interface{})
			newNodeMap := n.([]interface{})[0].(map[string]interface{})

			if oldNodeMap["node_product_code"] != newNodeMap["node_product_code"] {
				return StringPtrOrNil(newNodeMap["node_product_code"], true)
			}
		}
	}
	return nil
}

func resourceNcloudCDSSClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)

	if err := waitForCDSSClusterActive(ctx, d, config, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	LogCommonRequest("resourceNcloudCDSSClusterDelete", d.Id())
	if _, _, err := config.Client.Vcdss.V1Api.ClusterDeleteCDSSClusterServiceGroupInstanceNoDelete(ctx, d.Id()); err != nil {
		LogErrorResponse("resourceNcloudCDSSClusterDelete", err, d.Id())
		return diag.FromErr(err)
	}

	if err := waitForCDSSClusterDeletion(ctx, d, config); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func waitForCDSSClusterDeletion(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{CDSSStatusDeleting},
		Target:  []string{CDSSStatusReturn},
		Refresh: func() (result interface{}, state string, err error) {
			cluster, err := getCDSSCluster(ctx, config, d.Id())
			if err != nil {
				return nil, "", err
			}
			if cluster == nil {
				return d.Id(), CDSSStatusNull, nil
			}
			return cluster, cluster.Status, nil
		},
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 3 * time.Second,
		Delay:      2 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("Error waiting for VCDSS Cluster (%s) to become terminating: %s", d.Id(), err)
	}
	return nil
}

func waitForCDSSClusterActive(ctx context.Context, d *schema.ResourceData, config *conn.ProviderConfig, id string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{CDSSStatusCreating, CDSSStatusChanging},
		Target:  []string{CDSSStatusRunning},
		Refresh: func() (result interface{}, state string, err error) {
			cluster, err := getCDSSCluster(ctx, config, id)
			if err != nil {
				return nil, "", err
			}
			if cluster == nil {
				return id, CDSSStatusNull, nil
			}
			return cluster, cluster.Status, nil
		},
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 3 * time.Second,
		Delay:      2 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("error waiting for CDSS Cluster (%s) to become activating: %s", id, err)
	}
	return nil
}

func getCDSSCluster(ctx context.Context, config *conn.ProviderConfig, id string) (*vcdss.OpenApiGetClusterInfoResponseVo, error) {
	resp, _, err := config.Client.Vcdss.V1Api.ClusterGetClusterInfoListServiceGroupInstanceNoPost(ctx, id)
	if err != nil {
		return nil, err
	}
	LogResponse("getCDSSCluster", resp)

	return resp.Result, nil
}

func getBrokerInfo(ctx context.Context, config *conn.ProviderConfig, id string) (*vcdss.GetBrokerNodeListsResponseVo, error) {
	resp, _, err := config.Client.Vcdss.V1Api.ClusterGetBrokerInfoServiceGroupInstanceNoGet(ctx, id)
	if err != nil {
		return nil, err
	}
	LogResponse("getBrokerInfo", resp)

	return resp.Result, nil
}
