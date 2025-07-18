package autoscaling_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/acctest"
)

func TestAccDataSourceNcloudLaunchConfiguration_vpc_basic(t *testing.T) {
	dataName := "data.ncloud_launch_configuration.lc"
	resourceName := "ncloud_launch_configuration.lc"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { TestAccPreCheck(t) },
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNcloudLaunchConfigurationVpcConfig(),
				Check: resource.ComposeTestCheckFunc(
					TestAccCheckDataSourceID(dataName),
					resource.TestCheckResourceAttrPair(dataName, "launch_configuration_no", resourceName, "launch_configuration_no"),
					resource.TestCheckResourceAttrPair(dataName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataName, "server_image_product_code", resourceName, "server_image_product_code"),
					resource.TestCheckResourceAttrPair(dataName, "server_product_code", resourceName, "server_product_code"),
					resource.TestCheckResourceAttrPair(dataName, "member_server_image_no", resourceName, "member_server_image_no"),
					resource.TestCheckResourceAttrPair(dataName, "login_key_name", resourceName, "login_key_name"),
					resource.TestCheckResourceAttrPair(dataName, "is_encrypted_volume", resourceName, "is_encrypted_volume"),
					resource.TestCheckResourceAttrPair(dataName, "init_script_no", resourceName, "init_script_no"),
				),
			},
		},
	})
}

func testAccDataSourceNcloudLaunchConfigurationVpcConfig() string {
	return `
resource "ncloud_launch_configuration" "lc" {
	server_image_product_code = "SW.VSVR.OS.LNX64.ROCKY.0810.B050"
	server_product_code = "SVR.VSVR.STAND.C002.M008.NET.SSD.B050.G002"
}

data "ncloud_launch_configuration" "lc" {
	id = ncloud_launch_configuration.lc.launch_configuration_no
}
`
}
