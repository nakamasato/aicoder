package applier

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
)

func TestUpdateBlock(t *testing.T) {
	originalContent := []byte(`resource "google_storage_bucket" "example_bucket" {
  name     = "example-bucket"
  location = "US"
}

resource "google_compute_instance" "unchanged_instance" {
  name         = "unchanged-instance"
  machine_type = "n1-standard-1"
  zone         = "us-central1-a"
}
`)
	f, diags := hclwrite.ParseConfig(originalContent, "example.tf", hcl.InitialPos)
	assert.False(t, diags.HasErrors(), "failed to parse HCL")

	err := UpdateBlock(f, "resource", "google_storage_bucket,example_bucket", `
name     = "new-example-bucket"
location = "EU"
`, nil)
	assert.Nil(t, err)

	expectedContent := `resource "google_storage_bucket" "example_bucket" {
  name     = "new-example-bucket"
  location = "EU"
}

resource "google_compute_instance" "unchanged_instance" {
  name         = "unchanged-instance"
  machine_type = "n1-standard-1"
  zone         = "us-central1-a"
}
`
	assert.Equal(t, expectedContent, string(f.Bytes()))
}

// NewComment is not implemented yet for HCL.
func TestUpdateBlockWithComment(t *testing.T) {
	// Initial HCL content
	initialContent := `// This is a comment
resource "example" "test" {
  name = "old_name"
}
`

	// New content to update the block
	newContent := `
name = "new_name"
`

	// New comments to add
	newComments := []string{
		"This is a new comment",
		"Another comment",
	}

	// Parse the initial content into an HCL file
	file, diags := hclwrite.ParseConfig([]byte(initialContent), "", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("failed to parse initial content: %s", diags.Error())
	}

	// Call the UpdateBlock function
	err := UpdateBlock(file, "resource", "example,test", newContent, newComments)
	if err != nil {
		t.Fatalf("UpdateBlock failed: %s", err)
	}

	// Expected output after update
	expectedOutput := `// This is a comment
resource "example" "test" {
  name = "new_name"
}
`

	// Compare the updated file content with the expected output
	if string(file.Bytes()) != expectedOutput {
		t.Errorf("unexpected output:\n%s\nexpected:\n%s", file.Bytes(), expectedOutput)
	}
}

func TestUpdateBlockModule(t *testing.T) {
	originalContent := []byte(`module "test" {
  source  = "./mod/test"
  members = [
    "user1",
    "user2",
  ]
}
`)
	f, diags := hclwrite.ParseConfig(originalContent, "example.tf", hcl.InitialPos)
	assert.False(t, diags.HasErrors(), "failed to parse HCL")

	err := UpdateBlock(f, "module", "test", `
  source  = "./mod/test"
  members = [
    "user1",
    "user2",
    "user3",
  ]
`, nil)
	assert.Nil(t, err)

	expectedContent := `module "test" {
  source = "./mod/test"
  members = [
    "user1",
    "user2",
    "user3",
  ]
}
`
	assert.Equal(t, expectedContent, string(f.Bytes()))
}
