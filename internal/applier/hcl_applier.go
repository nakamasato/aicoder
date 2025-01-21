package applier

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/nakamasato/aicoder/internal/planner"
)

type hclApplier struct{}

func (a *hclApplier) Apply(r io.Reader, c planner.BlockChange) ([]byte, error) {

	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}
	f, diag := hclwrite.ParseConfig(src, c.Block.Path, hcl.InitialPos)

	if diag.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file: %s, error: %v", c.Block.Path, diag.Error())
	}
	err = UpdateBlock(f, c.Block.TargetType, c.Block.TargetName, c.NewContent, nil) // targetname is strings.Join(block.Labels(), ",") and newComments is not implemented yet
	if err != nil {
		return nil, fmt.Errorf("failed to update block (%s): %w", c.Block.Path, err)
	}
	return f.Bytes(), nil
}

// updateBlock replaces the entire content of the specified resource block with new content
func UpdateBlock(f *hclwrite.File, blockType, resourceName string, newContent string, newComments []string) error {
	body := f.Body()
	blocks := body.Blocks()

	for i, block := range blocks {
		log.Println(i, " blockType: ", blockType, ", resourceName: ", resourceName, ", block.Type: ", block.Type(), ", block.Labels: ", strings.Join(block.Labels(), ","))
		if block.Type() == blockType && strings.Join(block.Labels(), ",") == resourceName {
			log.Println("found matched block for ", resourceName)
			// Parse the new content into a temporary HCL file
			tempFile, diags := hclwrite.ParseConfig([]byte(newContent), "", hcl.InitialPos)
			if diags.HasErrors() {
				log.Fatalf("failed to parse new block content: %s", diags.Error())
			}

			// Clear the existing block body and append new tokens
			block.Body().Clear()
			block.Body().AppendUnstructuredTokens(tempFile.Body().BuildTokens(nil))

			// Add new comments
			if len(newComments) > 0 {
				fmt.Println("newComments is not implemented yet for HCL.")
			}
			return nil
		}
	}
	return fmt.Errorf("resource block not found: %s", resourceName)
}
