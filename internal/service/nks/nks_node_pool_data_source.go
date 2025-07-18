package nks

import (
	"context"
	"log"
	"strconv"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/terraform-providers/terraform-provider-ncloud/internal/conn"
)

func DataSourceNcloudNKSNodePool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNcloudNKSNodePoolRead,
		Schema: map[string]*schema.Schema{
			"cluster_uuid": {
				Type:     schema.TypeString,
				Required: true,
			},
			"instance_no": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"k8s_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"node_pool_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"node_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"subnet_no": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "use 'subnet_no_list' instead",
			},
			"subnet_no_list": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"product_code": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"software_code": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_size": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"server_spec_code": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"server_role_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"autoscale": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"max": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"min": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"label": {
				Type:       schema.TypeSet,
				Optional:   true,
				ConfigMode: schema.SchemaConfigModeAttr,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"taint": {
				Type:       schema.TypeSet,
				Optional:   true,
				ConfigMode: schema.SchemaConfigModeAttr,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"effect": {
							Type:     schema.TypeString,
							Required: true,
						},
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"nodes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instance_no": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"spec": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"private_ip": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"public_ip": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"node_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"container_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"kernel_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceNcloudNKSNodePoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*conn.ProviderConfig)
	clusterUuid := d.Get("cluster_uuid").(string)
	nodePoolName := d.Get("node_pool_name").(string)
	id := NodePoolCreateResourceID(clusterUuid, nodePoolName)

	nodePool, err := GetNKSNodePool(ctx, config, clusterUuid, nodePoolName)
	if err != nil {
		return diag.FromErr(err)
	}

	if nodePool == nil {
		d.SetId("")
		return nil
	}

	d.SetId(id)

	d.Set("cluster_uuid", clusterUuid)
	d.Set("instance_no", strconv.Itoa(int(ncloud.Int32Value(nodePool.InstanceNo))))
	d.Set("node_pool_name", nodePool.Name)
	d.Set("product_code", nodePool.ProductCode)
	d.Set("software_code", nodePool.SoftwareCode)
	d.Set("node_count", nodePool.NodeCount)
	d.Set("k8s_version", nodePool.K8sVersion)
	d.Set("server_spec_code", nodePool.ServerSpecCode)
	d.Set("storage_size", strconv.Itoa(int(ncloud.Int32Value(nodePool.StorageSize))))
	d.Set("server_role_id", nodePool.ServerRoleId)

	if len(nodePool.SubnetNoList) > 0 {
		if err := d.Set("subnet_no_list", flattenInt32ListToStringList(nodePool.SubnetNoList)); err != nil {
			log.Printf("[WARN] Error setting subnet no list set for (%s): %s", d.Id(), err)
		}
	}

	if err := d.Set("autoscale", flattenNKSNodePoolAutoScale(nodePool.Autoscale)); err != nil {
		log.Printf("[WARN] Error setting Autoscale set for (%s): %s", d.Id(), err)
	}

	if err := d.Set("taint", flattenNKSNodePoolTaints(nodePool.Taints)); err != nil {
		log.Printf("[WARN] Error setting taints set for (%s): %s", d.Id(), err)
	}

	if err := d.Set("label", flattenNKSNodePoolLabels(nodePool.Labels)); err != nil {
		log.Printf("[WARN] Error setting labels set for (%s): %s", d.Id(), err)
	}

	nodes, err := getNKSNodePoolWorkerNodes(ctx, config, clusterUuid, nodePoolName)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("nodes", flattenNKSWorkerNodes(nodes)); err != nil {
		log.Printf("[WARN] Error setting workerNodes set for (%s): %s", d.Id(), err)
	}
	return nil
}
