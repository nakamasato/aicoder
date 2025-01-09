package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHCL(t *testing.T) {
	// 一時ディレクトリと HCL ファイルを作成
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
		t.Fatalf("Failed to write hcl file: %v", err)
	}

	blocks, attrs, err := ParseHCL(hclPath)
	if err != nil {
		t.Fatalf("ParseHCL returned error: %v", err)
	}

	// 期待されるブロックと属性が見つかっているかを確認
	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(blocks))
	}

	if len(attrs) == 0 {
		t.Errorf("Expected attributes, got 0")
	}

	foundVariable := false
	foundResource := false
	for _, b := range blocks {
		if b.Type == "variable" {
			foundVariable = true
		}
		if b.Type == "resource" {
			foundResource = true
		}
	}
	if !foundVariable {
		t.Errorf("Did not find variable block")
	}
	if !foundResource {
		t.Errorf("Did not find resource block")
	}

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
