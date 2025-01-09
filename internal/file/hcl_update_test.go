package file

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
)

func TestUpdateAttributes(t *testing.T) {
	originalContent := []byte(`resource "google_secret_manager_secret_iam_member" "example_sa_is_slack_token_secret_accessor" {
  project   = var.gcp_project_id
  member    = google_service_account.example_sa.member
  secret_id = google_secret_manager_secret.slack_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
}

resource "google_storage_bucket" "unchanged_bucket" {
  name     = "unchanged-bucket"
  location = "US"
}
`)
	f, diags := hclwrite.ParseConfig(originalContent, "example.tf", hcl.InitialPos)
	assert.False(t, diags.HasErrors(), "failed to parse HCL")

	UpdateAttributes(f, "example_sa_is_slack_token_secret_accessor", map[string]string{
		"role": "roles/secretmanager.secretAdmin",
	})

	expectedContent := `resource "google_secret_manager_secret_iam_member" "example_sa_is_slack_token_secret_accessor" {
  project   = var.gcp_project_id
  member    = google_service_account.example_sa.member
  secret_id = google_secret_manager_secret.slack_token.secret_id
  role      = "roles/secretmanager.secretAdmin"
}

resource "google_storage_bucket" "unchanged_bucket" {
  name     = "unchanged-bucket"
  location = "US"
}
`
	assert.Equal(t, expectedContent, string(f.Bytes()))
}

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

	UpdateBlock(f, "example_bucket", `
name     = "new-example-bucket"
location = "EU"
`)

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

func TestAddBlock(t *testing.T) {
	originalContent := []byte(`resource "google_storage_bucket" "unchanged_bucket" {
  name     = "unchanged-bucket"
  location = "US"
}
`)
	f, diags := hclwrite.ParseConfig(originalContent, "example.tf", hcl.InitialPos)
	assert.False(t, diags.HasErrors(), "failed to parse HCL")

	AddBlock(f, "resource", []string{"google_compute_instance", "example_instance"}, `  name         = "example-instance"
  machine_type = "n1-standard-1"
  zone         = "us-central1-a"
`)

	expectedContent := `resource "google_storage_bucket" "unchanged_bucket" {
  name     = "unchanged-bucket"
  location = "US"
}
resource "google_compute_instance" "example_instance" {
  name         = "example-instance"
  machine_type = "n1-standard-1"
  zone         = "us-central1-a"
 }
`
	assert.Equal(t, expectedContent, string(f.Bytes()))
}

func TestUpdateResourceNames(t *testing.T) {
	originalContent := []byte(`resource "google_compute_instance" "example_instance" {
  name         = "example-instance"
  machine_type = "n1-standard-1"
  zone         = "us-central1-a"
}

resource "google_storage_bucket" "unchanged_bucket" {
  name     = "unchanged-bucket"
  location = "US"
}
`)
	f, diags := hclwrite.ParseConfig(originalContent, "example.tf", hcl.InitialPos)
	assert.False(t, diags.HasErrors(), "failed to parse HCL")

	UpdateResourceNames(f, map[string]string{"example_instance": "new_example_instance"})

	expectedContent := `resource "google_compute_instance" "new_example_instance" {
  name         = "example-instance"
  machine_type = "n1-standard-1"
  zone         = "us-central1-a"
}

resource "google_storage_bucket" "unchanged_bucket" {
  name     = "unchanged-bucket"
  location = "US"
}
`
	assert.Equal(t, expectedContent, string(f.Bytes()))
}
