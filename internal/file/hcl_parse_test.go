package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHCL(t *testing.T) {
	// Create a temporary directory and HCL file
	tempDir, err := os.MkdirTemp("", "hcltest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hclContent := `
variable "example" {
  default = "value"
}

resource "aws_instance" "example" {
  ami           = "ami-123456"
  instance_type = "t2.micro"
}
`
	hclPath := filepath.Join(tempDir, "test.hcl")
	err = os.WriteFile(hclPath, []byte(hclContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write HCL file: %v", err)
	}

	blocks, attrs, err := ParseHCL(hclPath)
	if err != nil {
		t.Fatalf("ParseHCL returned error: %v", err)
	}

	// Verify that the expected blocks and attributes are found
	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(blocks))
	}

	if len(attrs) == 0 {
		t.Errorf("Expected attributes, got 0")
	}

	foundVariable := false
	foundResource := false

	for _, b := range blocks {
		switch b.Type {
		case "variable":
			foundVariable = true
		case "resource":
			foundResource = true
		}
	}
	if !foundVariable {
		t.Errorf("Did not find variable block")
	}
	if !foundResource {
		t.Errorf("Did not find resource block")
	}

	// For attribute 'ami', check start and end lines

	foundAMI := false
	for _, attr := range attrs {
		if attr.Name == "ami" {
			foundAMI = true
			break
		}
	}
	if !foundAMI {
		t.Errorf("Did not find attribute 'ami'")
	}
}

func TestGoogleSecretManagerSecretIAMMember(t *testing.T) {
	// Create a temporary directory and HCL file with the new resource example
	tempDir, err := os.MkdirTemp("", "hcltest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hclContent := `
resource "google_secret_manager_secret_iam_member" "example_sa_is_slack_token_secret_accessor" {
  project   = var.gcp_project_id
  member    = google_service_account.example_sa.member
  secret_id = google_secret_manager_secret.slack_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
}
`
	hclPath := filepath.Join(tempDir, "google_secret_manager.hcl")
	err = os.WriteFile(hclPath, []byte(hclContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write HCL file: %v", err)
	}

	blocks, _, err := ParseHCL(hclPath)
	if err != nil {
		t.Fatalf("ParseHCL returned error: %v", err)
	}

	// Verify that the expected block is found
	foundResource := false

	for _, b := range blocks {
		if b.Type == "resource" && b.Labels[0] == "google_secret_manager_secret_iam_member" {
			foundResource = true
			break
		}
	}
	if !foundResource {
		t.Errorf("Did not find resource block 'google_secret_manager_secret_iam_member'")
	}
}

func TestGoogleSecretManagerSecretIAMMemberAttributes(t *testing.T) {
	// Create a temporary directory and HCL file with the new resource example
	tempDir, err := os.MkdirTemp("", "hcltest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hclContent := `
resource "google_secret_manager_secret_iam_member" "example_sa_is_slack_token_secret_accessor" {
  project   = var.gcp_project_id
  member    = google_service_account.example_sa.member
  secret_id = google_secret_manager_secret.slack_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
}
`
	hclPath := filepath.Join(tempDir, "google_secret_manager.tf")
	err = os.WriteFile(hclPath, []byte(hclContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write HCL file: %v", err)
	}

	_, attrs, err := ParseHCL(hclPath)
	if err != nil {
		t.Fatalf("ParseHCL returned error: %v", err)
	}

	// Verify that the expected attributes are found
	expectedAttributes := map[string]string{
		"project":   "var.gcp_project_id",
		"member":    "google_service_account.example_sa.member",
		"secret_id": "google_secret_manager_secret.slack_token.secret_id",
		"role":      "\"roles/secretmanager.secretAccessor\"",
	}

	for name, expectedValue := range expectedAttributes {
		found := false
		for _, attr := range attrs {
			if attr.Name == name {
				found = true
				if attr.Value != expectedValue {
					t.Errorf("Expected %s attribute value %s, got %s", name, expectedValue, attr.Value)
				}
				break
			}
		}
		if !found {
			t.Errorf("Did not find attribute %s", name)
		}
	}
}
