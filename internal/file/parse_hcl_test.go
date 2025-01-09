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

	// Expected line ranges based on the content
	expectedVariableStart := 2
	expectedVariableEnd := 4
	expectedResourceStart := 6
	expectedResourceEnd := 9

	for _, b := range blocks {
		switch b.Type {
		case "variable":
			foundVariable = true
			if b.StartLine != expectedVariableStart {
				t.Errorf("Expected variable block start line %d, got %d", expectedVariableStart, b.StartLine)
			}
			if b.EndLine != expectedVariableEnd {
				t.Errorf("Expected variable block end line %d, got %d", expectedVariableEnd, b.EndLine)
			}
		case "resource":
			foundResource = true
			if b.StartLine != expectedResourceStart {
				t.Errorf("Expected resource block start line %d, got %d", expectedResourceStart, b.StartLine)
			}
			if b.EndLine != expectedResourceEnd {
				t.Errorf("Expected resource block end line %d, got %d", expectedResourceEnd, b.EndLine)
			}
		}
	}
	if !foundVariable {
		t.Errorf("Did not find variable block")
	}
	if !foundResource {
		t.Errorf("Did not find resource block")
	}

	// For attribute 'ami', check start and end lines
	expectedAMIStart := 7
	expectedAMIEnd := 7

	foundAMI := false
	for _, attr := range attrs {
		if attr.Name == "ami" {
			foundAMI = true
			if attr.StartLine != expectedAMIStart {
				t.Errorf("Expected 'ami' attribute start line %d, got %d", expectedAMIStart, attr.StartLine)
			}
			if attr.EndLine != expectedAMIEnd {
				t.Errorf("Expected 'ami' attribute end line %d, got %d", expectedAMIEnd, attr.EndLine)
			}
			break
		}
	}
	if !foundAMI {
		t.Errorf("Did not find attribute 'ami'")
	}
}


func TestAttributeMultilineEndLine(t *testing.T) {
	// Create a temporary directory and HCL file with a multiline attribute
	tempDir, err := os.MkdirTemp("", "hcltest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hclContent := `
resource "example_resource" "test" {
  description = <<EOF
This is a multiline
description attribute.
It spans multiple lines.
EOF
}
`
	hclPath := filepath.Join(tempDir, "multiline.hcl")
	err = os.WriteFile(hclPath, []byte(hclContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write HCL file: %v", err)
	}

	_, attrs, err := ParseHCL(hclPath)
	if err != nil {
		t.Fatalf("ParseHCL returned error: %v", err)
	}

	// Find the 'description' attribute and verify its end line
	foundDescription := false
	expectedEndLine := 7 // EOF should be on line 7 based on the hclContent above
	for _, attr := range attrs {
		if attr.Name == "description" {
			foundDescription = true
			if attr.EndLine != expectedEndLine {
				t.Errorf("Expected 'description' attribute end line %d, got %d", expectedEndLine, attr.EndLine)
			}
			break
		}
	}
	if !foundDescription {
		t.Errorf("Did not find attribute 'description'")
	}
}
