package devtools_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vsourcedeploy"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/acctest"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/conn"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/service/devtools"
)

func TestAccResourceNcloudSourceDeployProject_basic(t *testing.T) {
	var project vsourcedeploy.GetIdNameResponse
	name := getTestSourceDeployProjectName()
	resourceName := "ncloud_sourcedeploy_project.test-project"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { TestAccPreCheck(t) },
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSourceDeployProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNcloudSourceDeployProjectConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSourceDeployProjectExists(resourceName, &project),
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccResourceNcloudSourceDeployProjectConfig(name string) string {
	return fmt.Sprintf(`
resource "ncloud_sourcedeploy_project" "test-project" {
	name = "%[1]s"
}
`, name)
}

func testAccCheckSourceDeployProjectExists(n string, project *vsourcedeploy.GetIdNameResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := TestAccProvider.Meta().(*conn.ProviderConfig)
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No project no is set")
		}
		resp, err := devtools.GetSourceDeployProjectById(context.Background(), config, rs.Primary.ID)
		if err != nil {
			return err
		}
		project = resp
		return nil
	}
}

func testAccCheckSourceDeployProjectDestroy(s *terraform.State) error {
	config := TestAccProvider.Meta().(*conn.ProviderConfig)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ncloud_sourcedeploy_project" {
			continue
		}

		resp, err := devtools.GetSourceDeployProjectById(context.Background(), config, rs.Primary.ID)

		if err != nil {
			return err
		}

		if resp != nil {
			return errors.New("project still exists")
		}
	}

	return nil
}

func getTestSourceDeployProjectName() string {
	rInt := acctest.RandIntRange(1, 9999)
	testProjectName := fmt.Sprintf("tf-%d-project", rInt)
	return testProjectName
}
