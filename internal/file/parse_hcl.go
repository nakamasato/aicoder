package file

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Block represents an HCL block with start and end line information.
type Block struct {
	Type      string
	Labels    []string
	StartLine int
	EndLine   int
}

// Attribute represents an HCL attribute with start and end line information.
type Attribute struct {
	Name      string
	StartLine int
	EndLine   int
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

// traverseBody recursively traverses the HCL body, extracting blocks and attributes with line ranges.
func traverseBody(body *hclsyntax.Body, blocks *[]Block, attrs *[]Attribute) {
	for _, block := range body.Blocks {
		defRange := block.DefRange
		endLine := block.Body.SrcRange.End.Line
		*blocks = append(*blocks, Block{
			Type:      block.Type,
			Labels:    block.Labels,
			StartLine: defRange().Start.Line,
			EndLine:   endLine,
		})
		traverseBody(block.Body, blocks, attrs)
	}
	for name, attr := range body.Attributes {
		srcRange := attr.SrcRange
		*attrs = append(*attrs, Attribute{
			Name:      name,
			StartLine: srcRange.Start.Line,
			EndLine:   srcRange.End.Line,
		})
	}
}
