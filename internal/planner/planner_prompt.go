package planner

const NECESSARY_CHANGES_PLAN_PROMPT = `Please make a plan to achieve the goal.
A plan consists of a series of steps to execute in order.

-----------------------
Goal: %s

-----------------------
Relevant files:
%s

-----------------------
Example:

1. Need to update [Function] in [File] to support [Feature].
2. Need to create [File] and implement [Function] to support [Feature].
3. Call [Function] created in step 2 in [File].
etc.

Each step will be corresponding to one change in a function.
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

Just provide the content of the block, not including the block name and lables.
`

const PLANNER_LINE_NUM_PROMPT = `Please provide the start and end line number of the target location.

## Target location

target_type: %s
target_name: %s

## File Content

` + "```\n" + `
%s
` + "```\n" + `

## Examples

target_type: function
target_name: NewPlanner

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

{\"start_line\":25,\"end_line\":30}

Reason:
---
1 package planner
2
3 import (
...
...
25 func NewPlanner(llmClient llm.Client, entClient *ent.Client) *Planner {
26 	return &Planner{
27 		llmClient: llmClient,
28		entClient: entClient,
29	}
30 }
---
`

const VALIDATE_GOAL_PROMPT = `Please validate the given goal.

Currently AICoder is still under development and only support goals that explicitly specficy a file to change or create.

-----------------------
Goal: %s`

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
