package server

import (
	"fmt"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/common"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/conn"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/verify"
)

func DataSourceNcloudServerImage() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNcloudServerImageRead,

		Schema: map[string]*schema.Schema{
			"product_code": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"platform_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"infra_resource_detail_type_code": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"filter": DataSourceFiltersSchema(),

			"product_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"product_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"product_description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"infra_resource_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"base_block_storage_size": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"os_information": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceNcloudServerImageRead(d *schema.ResourceData, meta interface{}) error {
	resources, err := getServerImageProductListFiltered(d, meta.(*conn.ProviderConfig))

	if err != nil {
		return err
	}

	if err := verify.ValidateOneResult(len(resources)); err != nil {
		return err
	}

	SetSingularResourceDataFromMap(d, resources[0])

	return nil
}

func getServerImageProductListFiltered(d *schema.ResourceData, config *conn.ProviderConfig) ([]map[string]interface{}, error) {
	var resources []map[string]interface{}
	var err error

	resources, err = getVpcServerImageProductList(d, config)
	if err != nil {
		return nil, err
	}

	if f, ok := d.GetOk("filter"); ok {
		resources = ApplyFilters(f.(*schema.Set), resources, DataSourceNcloudServerImage().Schema)
	}

	return resources, nil
}

func getVpcServerImageProductList(d *schema.ResourceData, config *conn.ProviderConfig) ([]map[string]interface{}, error) {
	client := config.Client
	regionCode := config.RegionCode

	reqParams := &vserver.GetServerImageProductListRequest{
		ProductCode:                 StringPtrOrNil(d.GetOk("product_code")),
		RegionCode:                  &regionCode,
		InfraResourceDetailTypeCode: StringPtrOrNil(d.GetOk("infra_resource_detail_type_code")),
	}

	if v, ok := d.GetOk("platform_type"); ok {
		reqParams.PlatformTypeCodeList = []*string{ncloud.String(v.(string))}
	}

	LogCommonRequest("GetServerImageProductList", reqParams)
	resp, err := client.Vserver.V2Api.GetServerImageProductList(reqParams)
	if err != nil {
		LogErrorResponse("GetServerImageProductList", err, reqParams)
		return nil, err
	}
	LogResponse("GetServerImageProductList", resp)

	var resources []map[string]interface{}

	for _, r := range resp.ProductList {
		instance := map[string]interface{}{
			"id":                      *r.ProductCode,
			"product_code":            *r.ProductCode,
			"product_name":            *r.ProductName,
			"product_type":            *r.ProductType.Code,
			"product_description":     *r.ProductDescription,
			"infra_resource_type":     *r.InfraResourceType.Code,
			"base_block_storage_size": fmt.Sprintf("%dGB", *r.BaseBlockStorageSize/GIGABYTE),
			"platform_type":           *r.PlatformType.Code,
			"os_information":          *r.OsInformation,
		}

		if r.InfraResourceDetailType != nil {
			instance["infra_resource_detail_type_code"] = *r.InfraResourceDetailType.Code
		}
		resources = append(resources, instance)
	}

	return resources, nil
}
