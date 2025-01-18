package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/openai/openai-go"
)

type ReviewResult struct {
	PlanId   string `json:"plan_id" jsonschema_description:"ID of the Plan to review"`
	Approved bool   `json:"result" jsonschema_description:"Whether the changeplan is approved or not"`
	Comment  string `json:"comment" jsonschema_description:"Overall review comment for the changeplan"`
}

var (
	ResultSchemaParam = llm.GenerateJsonSchemaParam[ReviewResult]("result", "Review result for the change plan")
)

const (
	REVIEW_PROMPT = `Goal: %s

Planned changes to review (PlanID: %s):
%s
--- Review Point ---

Please give your comment for changes.
If you think the change is necessary and reasonable to achieve the goal, please leave a comment like "Looks good to me" with a reason.
If you think the change is not necessary nor reasonable, please include the reason and suggest an alternative if possible.
Please consider if the target file is correct and the change is necessary to achieve the goal.
If the changes are certain to achieve the goal, the result should be "true".
`
)

// ReviewChanges applies changes based on the provided changesPlan.
// If dryrun is true, it displays the diffs without modifying the actual files.
func ReviewChanges(ctx context.Context, llmClient llm.Client, changesPlan *planner.ChangesPlan) (*ReviewResult, error) {
	fmt.Println("Reviewing changes for query:", changesPlan.Query)

	message := fmt.Sprintf(REVIEW_PROMPT, changesPlan.Query, changesPlan.Id, makeChangeString(&changesPlan.Changes))

	content, err := llmClient.GenerateCompletion(ctx,
		[]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful reviewer to check whether the planned changes are reasonable to achieve goal. Please give your comment on each change."),
			openai.UserMessage(message),
		},
		ResultSchemaParam)

	if err != nil {
		return nil, fmt.Errorf("failed to generate completion: %w", err)
	}
	fmt.Println("Review result:", content)
	var result ReviewResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal review result: %w", err)
	}

	return &result, nil
}

func makeChangeString(changes *[]planner.BlockChange) string {
	var builder strings.Builder
	for i, change := range *changes {
		builder.WriteString(fmt.Sprintf(`---- change %d -----
Path: %s, Type: %s, Name: %s,
NewContent: %s
----- change %d end ----
`, i, change.Block.Path, change.Block.TargetType, change.Block.TargetName, change.NewContent, i))
	}
	changesString := builder.String()
	return changesString
}
