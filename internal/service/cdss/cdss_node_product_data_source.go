package cdss

import (
	"context"
	"fmt"
	"strconv"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vcdss"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/common"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/conn"
)

func DataSourceNcloudCDSSNodeProduct() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNcloudCDSSNodeProductRead,
		Schema: map[string]*schema.Schema{
			"os_image": {
				Type:     schema.TypeString,
				Required: true,
			},
			"subnet_no": {
				Type:     schema.TypeString,
				Required: true,
			},
			"filter": DataSourceFiltersSchema(),
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu_count": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"memory_size": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"product_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceNcloudCDSSNodeProductRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*conn.ProviderConfig)

	reqParams := vcdss.NodeProduct{
		SoftwareProductCode: *StringPtrOrNil(d.GetOk("os_image")),
		SubnetNo:            *GetInt32FromString(d.GetOk("subnet_no")),
	}

	resources, err := getCDSSNodeProducts(config, reqParams)
	if err != nil {
		return err
	}

	if f, ok := d.GetOk("filter"); ok {
		resources = ApplyFilters(f.(*schema.Set), resources, DataSourceNcloudCDSSNodeProduct().Schema)
	}

	if len(resources) < 1 {
		return fmt.Errorf("no results. please change search criteria and try again")
	}

	for k, v := range resources[0] {
		if k == "id" {
			d.SetId(v.(string))
		}
		d.Set(k, v)
	}

	return nil
}

func getCDSSNodeProducts(config *conn.ProviderConfig, reqParams vcdss.NodeProduct) ([]map[string]interface{}, error) {
	LogCommonRequest("GetOsProductList", reqParams)

	resp, _, err := config.Client.Vcdss.V1Api.ClusterGetNodeProductListPost(context.Background(), reqParams)
	if err != nil {
		LogErrorResponse("GetOsProductList", err, "")
		return nil, err
	}
	LogResponse("GetOsProductList", resp)

	resources := []map[string]interface{}{}

	for _, r := range resp.Result.ProductList {
		memorySize, err := parseMemorySize(r.MemorySize)
		if err != nil {
			LogErrorResponse("Invalid Memory Size", err, "")
			return nil, err
		}

		instance := map[string]interface{}{
			"id":           ncloud.StringValue(&r.ProductCode),
			"cpu_count":    ncloud.StringValue(&r.CpuCount),
			"memory_size":  ncloud.StringValue(memorySize),
			"product_type": ncloud.StringValue(&r.ProductType2Code),
		}

		resources = append(resources, instance)
	}

	return resources, nil
}

func parseMemorySize(memorySize string) (*string, error) {
	num, err := strconv.Atoi(memorySize)
	if err != nil {
		return nil, err
	}
	res := num / 1024 / 1024 / 1024
	resFormatGB := strconv.Itoa(res) + "GB"
	return &resFormatGB, err
}
