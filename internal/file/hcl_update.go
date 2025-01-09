package file

import (
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// updateAttributes updates specific attributes in the first resource block found
func UpdateAttributes(f *hclwrite.File, attrs map[string]string) {
	body := f.Body()
	blocks := body.Blocks()

	for _, block := range blocks {
		if block.Type() == "resource" {
			for attrName, attrValue := range attrs {
				if block.Body().GetAttribute(attrName) != nil {
					block.Body().SetAttributeValue(attrName, cty.StringVal(attrValue))
				}
			}
			break // Assuming you only want to update the first matching block
		}
	}
}

// updateResourceNames updates the names of all resource blocks found
func UpdateResourceNames(f *hclwrite.File, newNames map[string]string) {
	body := f.Body()
	blocks := body.Blocks()

	for _, block := range blocks {
		if block.Type() == "resource" {
			labels := block.Labels()
			if len(labels) > 1 {
				// Check if there's a new name for this resource
				if newName, exists := newNames[labels[1]]; exists {
					// Update the resource name (second label)
					labels[1] = newName
					// Directly update the block's labels without removing and re-adding
					block.SetLabels(labels)
				}
			}
		}
	}
}

// updateBlock replaces the entire content of the specified resource block with new content
func UpdateBlock(f *hclwrite.File, resourceName string, newContent string) {
	body := f.Body()
	blocks := body.Blocks()

	for _, block := range blocks {
		if block.Type() == "resource" {
			labels := block.Labels()
			if len(labels) > 1 && labels[1] == resourceName {
				// Parse the new content into a temporary HCL file
				tempFile, diags := hclwrite.ParseConfig([]byte(newContent), "", hcl.InitialPos)
				if diags.HasErrors() {
					log.Fatalf("failed to parse new block content: %s", diags.Error())
				}

				// Clear the existing block body and append new tokens
				block.Body().Clear()
				block.Body().AppendUnstructuredTokens(tempFile.Body().BuildTokens(nil))
				break // Assuming you only want to update the first matching block
			}
		}
	}
}
