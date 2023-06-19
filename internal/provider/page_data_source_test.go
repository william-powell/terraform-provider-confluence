package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPageDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "confluence_page" "test" {
  title = "Unit Test Page"
  parent_id = "33296"
  body = "<p>Unit Test Page</p>"
}

data "confluence_page" "test" {
	id = confluence_page.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the item to ensure all attributes are set
					resource.TestCheckResourceAttr("data.confluence_page.test", "body", "<p>Unit Test Page</p>"),
					resource.TestCheckResourceAttr("confluence_page.test", "parent_id", "33296"),
					resource.TestCheckResourceAttr("confluence_page.test", "title", "Unit Test Page"),
					resource.TestCheckResourceAttr("confluence_page.test", "body", "<p>Unit Test Page</p>"),

					// Verify placeholder id attribute
					resource.TestCheckResourceAttrSet("confluence_page.test", "id"),
				),
			},
		},
	})
}
