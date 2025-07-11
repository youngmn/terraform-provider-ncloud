package postgresql_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	. "github.com/terraform-providers/terraform-provider-ncloud/internal/acctest"
)

func TestAccDataSourceNcloudPostgresqlDatabases_vpc_basic(t *testing.T) {
	dataName := "data.ncloud_postgresql_databases.all"
	testPostgresqlName := fmt.Sprintf("tf-postgresql-%s", acctest.RandString(5))
	dbResourceName := "ncloud_postgresql.postgresql"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { TestAccPreCheck(t) },
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPostgresqlDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePostgresqlDatabasesConfig(testPostgresqlName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataName, "postgresql_database_list.0.name", "testdb1"),
					resource.TestCheckResourceAttr(dataName, "postgresql_database_list.1.name", "testdb2"),
				),
				Destroy: false,
			},
			{
				Config: testAccPostgresqlDatabasesRemoveConfig(testPostgresqlName),
				Check: resource.ComposeTestCheckFunc(
					testAccPostgresqlDatabasesNotExist(dbResourceName, []string{"testdb1", "testdb2"}, TestAccProvider),
				),
			},
		},
	})
}

func testAccDataSourcePostgresqlDatabasesConfig(testPostgresqlName string) string {
	return fmt.Sprintf(`
resource "ncloud_vpc" "test_vpc" {
	name               = "%[1]s"
	ipv4_cidr_block    = "10.5.0.0/16"
}
resource "ncloud_subnet" "test_subnet" {
	vpc_no             = ncloud_vpc.test_vpc.vpc_no
	name               = "%[1]s"
	subnet             = "10.5.0.0/24"
	zone               = "KR-2"
	network_acl_no     = ncloud_vpc.test_vpc.default_network_acl_no
	subnet_type        = "PUBLIC"
}

resource "ncloud_postgresql" "postgresql" {
	vpc_no            = ncloud_vpc.test_vpc.vpc_no
	subnet_no         = ncloud_subnet.test_subnet.id
	service_name      = "%[1]s"
	server_name_prefix = "testprefix"
	user_name         = "testusername"
	user_password     = "t123456789!a"
	client_cidr       = "0.0.0.0/0"
	database_name     = "test_db"
}

resource "ncloud_postgresql_users" "postgresql_users" {
	id = ncloud_postgresql.postgresql.id
	postgresql_user_list = [
		{
			name = "test1",
			password = "t123456789!",
			client_cidr = "0.0.0.0/0",
			replication_role = "false"
		},
		{
			name = "test2",
			password = "t123456789!",
			client_cidr = "0.0.0.0/0",
			replication_role = "false"
		}
	]
}

resource "ncloud_postgresql_databases" "postgresql_db" {
	id = ncloud_postgresql.postgresql.id
	postgresql_database_list = [
		{
			name = "testdb1",
			owner = ncloud_postgresql_users.postgresql_users.postgresql_user_list.0.name
		},
		{
			name = "testdb2",
			owner = ncloud_postgresql_users.postgresql_users.postgresql_user_list.1.name
		}
	]
}

data "ncloud_postgresql_databases" "all" {
	id = ncloud_postgresql.postgresql.id
	filter {
		name = "name"
		values = [ncloud_postgresql_databases.postgresql_db.postgresql_database_list.0.name, ncloud_postgresql_databases.postgresql_db.postgresql_database_list.1.name]
	}
}
`, testPostgresqlName)
}
