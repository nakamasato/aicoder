package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Block represents an HCL block with start and end line information.
type Block struct {
	Type    string
	Content string
	Labels  []string
}

// Attribute represents an HCL attribute with start and end line information.
type Attribute struct {
	Name  string
	Value string
}

// ParseHCL parses the specified HCL file and returns the blocks and attributes.
func ParseHCL(path string) ([]Block, []Attribute, error) {
	if filepath.Ext(path) != ".hcl" && filepath.Ext(path) != ".tf" {
		return nil, nil, fmt.Errorf("file is not an hcl file: %s", path)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to read file: %s, error: %v", path, err)
		return nil, nil, err
	}

	file, diag := hclwrite.ParseConfig(src, path, hcl.InitialPos)
	if diag.HasErrors() {
		fmt.Printf("Failed to parse HCL file: %s, error: %v", path, diag.Error())
		return nil, nil, fmt.Errorf("failed to parse HCL file: %s, error: %v", path, diag.Error())
	}

	var blocks []Block
	var attrs []Attribute

	traverseBody(file.Body(), &blocks, &attrs)

	return blocks, attrs, nil
}

// traverseBody recursively traverses the HCL body, extracting blocks and attributes with line ranges.
func traverseBody(body *hclwrite.Body, blocks *[]Block, attrs *[]Attribute) {
	for _, block := range body.Blocks() {
		tokens := block.Body().BuildTokens(nil)
		*blocks = append(*blocks, Block{
			Type:    block.Type(),
			Content: string(hclwrite.Format(tokens.Bytes())),
			Labels:  block.Labels(),
		})
		traverseBody(block.Body(), blocks, attrs)
	}
	for name, attr := range body.Attributes() {

		tokens := attr.Expr().BuildTokens(nil)
		var value string
		for _, t := range tokens {
			value += string(t.Bytes)
		}
		*attrs = append(*attrs, Attribute{
			Name:  name,
			Value: value,
		})
	}
}
