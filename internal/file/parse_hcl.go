package file

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Block represents an HCL block.
type Block struct {
	Type   string
	Labels []string
	Line   int
}

// Attribute represents an HCL attribute.
type Attribute struct {
	Name string
	Line int
}

// ParseHCL parses the specified HCL file and returns the blocks and attributes.
func ParseHCL(path string) ([]Block, []Attribute, error) {
	if filepath.Ext(path) != ".hcl" {
		return nil, nil, fmt.Errorf("file is not an hcl file: %s", path)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read file: %s, error: %v", path, err)
		return nil, nil, err
	}

	file, diag := hclsyntax.ParseConfig(src, path, hcl.InitialPos)
	if diag.HasErrors() {
		log.Printf("Failed to parse HCL file: %s, error: %v", path, diag.Error())
		return nil, nil, fmt.Errorf("failed to parse HCL file: %s, error: %v", path, diag.Error())
	}

	var blocks []Block
	var attrs []Attribute

	traverseBody(file.Body.(*hclsyntax.Body), &blocks, &attrs)

	return blocks, attrs, nil
}

// traverseBody recursively traverses the HCL body, extracting blocks and attributes.
func traverseBody(body *hclsyntax.Body, blocks *[]Block, attrs *[]Attribute) {
	for _, block := range body.Blocks {
		pos := block.DefRange().Start.Line
		*blocks = append(*blocks, Block{
			Type:   block.Type,
			Labels: block.Labels,
			Line:   pos,
		})
		traverseBody(block.Body, blocks, attrs)
	}
	for name, attr := range body.Attributes {
		pos := attr.SrcRange.Start.Line
		*attrs = append(*attrs, Attribute{
			Name: name,
			Line: pos,
		})
	}
}
