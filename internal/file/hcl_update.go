package file

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// updateAttributes updates specific attributes in the specified resource block
func UpdateAttributes(f *hclwrite.File, resourceName string, attrs map[string]string) {
	body := f.Body()
	blocks := body.Blocks()

	for _, block := range blocks {
		if block.Type() == "resource" {
			labels := block.Labels()
			if len(labels) > 1 && labels[1] == resourceName {
				for attrName, attrValue := range attrs {
					if block.Body().GetAttribute(attrName) != nil {
						block.Body().SetAttributeValue(attrName, cty.StringVal(attrValue))
					}
				}
				break // Assuming you only want to update the first matching block
			}
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

func GetBlockContent(f *hclwrite.File, resourceName string) (string, error) {
	body := f.Body()
	blocks := body.Blocks()

	for _, block := range blocks {
		if block.Type() == "resource" {
			labels := block.Labels()
			if len(labels) > 1 && labels[1] == resourceName {
				tokens := block.Body().BuildTokens(nil)
				return string(hclwrite.Format(tokens.Bytes())), nil
			}
		}
	}
	return "", fmt.Errorf("resource block not found: %s", resourceName)
}

// updateBlock replaces the entire content of the specified resource block with new content
func UpdateBlock(f *hclwrite.File, blockType, resourceName string, newContent string) error {
	body := f.Body()
	blocks := body.Blocks()

	for i, block := range blocks {
		log.Println(i, " blockType: ", blockType, " resourceName: ", resourceName, " block.Type: ", block.Type())
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
			return nil
		}
	}
	return fmt.Errorf("resource block not found: %s", resourceName)
}

// AddBlock adds a new block to the HCL file.
func AddBlock(f *hclwrite.File, blockType string, labels []string, blockContent string) {
	block := hclwrite.NewBlock(blockType, labels)
	body := block.Body()
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(blockContent),
		},
	})
	f.Body().AppendBlock(block)
}
