package planner

const NECESSARY_CHANGES_PLAN_PROMPT = `Please make a plan to achieve the goal.
A plan consists of a series of steps to execute in order. Currently possible action is to replace content of a block in a file.
You can make several steps (changing multiple blocks in multiple files if necessary) to achieve the goal.

-----------------------
Goal: %s

-----------------------
Relevant files:
%s

Examples:
--------- Example 1 Start --------------
Goal: Please add tanaka@gmail.com and john@gmail.com to the member as analytics members to data-user.

Files:

terraform/team/data_users.tf:

` + "```\n" + `
module "data-user" {
  source  = "../modules/team"
  members = [
    // Manager
    "sudo@gmail.com",
	// Developer
	"hiroshi@gmail.com",
	"ken@gmail.com",
	// Analytics
	"shiho@gmail.com",
 ]
}
` + "```\n" + `

Steps:
1. Add tanaka@gmail.com and john@gmail.com to the members list after shiho@gmail.com in the data-user module in the file terraform/team/data_users.tf.

--------- Example 1 End --------------

etc.

Ideally each step will be corresponding to one change in a block.
`

const PLANNER_EXTRACT_BLOCK_FOR_STEP_PROMPT = `You are a helpful assistant that extract blocks to execute the given step.
Please consider necessary changes to do the given step.
-----------------------
Step: %s

-----------------------
Files option:
%s

------------------------
Please provide the complete set of locations as either a class name, a function name, a struct name, or a variable name.
Event if multiple files are provided, not necessarily all files need to be changed. Please only provide the blocks that need to be changed.


### Examples:

Code:

` + "```\n" + `
package planner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/invopop/jsonschema"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/openai/openai-go"
)

type Planner struct {
	llmClient llm.Client
	entClient *ent.Client
}

func NewPlanner(llmClient llm.Client, entClient *ent.Client) *Planner {
	return &Planner{
		llmClient: llmClient,
		entClient: entClient,
	}
}
` + "```\n" + `

Output:

{\"path\":\"internal/planner/planner.go\",\"target_type\":\"function\",\"target_name\":\"NewPlanner\"}

------------------------
`

const GENERATE_FUNCTION_CHANGES_PLAN_PROMPT_GO = `Please provide the new content of the Go function '%s' in the file '%s'
## Current content

`+"```"+`
%s
`+"```"+`

Note that please do not include the function signature in the new content.

Output Example:
`+"```"+`
fmt.Println("Hello, World!")
`+"```"+`
`

const GENERATE_BLOCK_CHANGES_PLAN_PROMPT_HCL = `Please provide the new content of the HCL block %s in the file %s

## Current content

`+"```"+`
%s
`+"```"+`

Example:

To replace the content of a block like this

`+"```"+`
resource "google_storage_bucket" "example_bucket" {
  name     = "example-bucket"
  location = "US"
}
`+"```"+`

Output should be like this:

`+"```"+`
name     = "new-example-bucket"
location = "EU"
`+"```"+`


Rules:
- Only provide the content of the block, not including the block name and lables.
- Make sure new line is added at the beginning and end of the content.
- Please do change unrelated content.
- If there's no need to change the content, please provide the current content.
`

const REPLAN_PROMPT = `You are a helpful assistant that generates detailed action plans based on provided project information.
The plan you've just made failed the validation.
Please provide a new plan based on the provided feedback.

-----------------------
Goal: %s
-----------------------
Previous plan:
%s
-----------------------
Previous errors:
%s
-----------------------

Multiple changes cannot be made for the same file. If you need multiple changes on the file. please update the target lines by adding and deleteing the content in the target lines.

Note that only adding content will have duplicated contents (title etc).

Example: 'update title in the readme'

Add: '## New Title'
Delete: '## Old Title'
Line: 1
`

const VALIDATE_FILE_PROMPT = `Please check the syntax of the file you have just modified.

The file content:
`
