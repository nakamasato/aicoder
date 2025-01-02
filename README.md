# AICoder## Change Plan Management
- `aicoder changeplan add`: Add a new change plan.
- `aicoder changeplan update <ID>`: Update an existing change plan.
- `aicoder changeplan delete <ID>`: Delete a change plan.


## Change Plan to Meet User Requirements

### Objective
Implement a feature to create a change plan based on user requirements for the AICoder tool, so users can efficiently manage and plan development tasks.

### Steps to Implement Feature

1. **Understand User Requirements**:
   - Conduct user interviews or surveys to gather requirements on how users want to plan their changes.

2. **Define Data Structure**:
   - Create a new structure in the application to hold change plans. This may include fields such as `Title`, `Description`, `Status`, and `AssignedTo`.
   - File to Modify: `internal/load/load.go`
   - Add a new struct in `load.go`:
     ```go
     type ChangePlan struct {
         Title       string
         Description string
         Status      string
         AssignedTo  string
     }
     ```

3. **Storage Mechanism**:
   - Decide where change plans will be stored (in databases, JSON files, etc.).
   - Potentially extend the database schema in `ent/schema/document.go` to include a `ChangePlan` entity.

4. **Implement Command-Line Interface (CLI) Commands**:
   - Create commands to add, update, delete, and view change plans.
   - File to Create: `cmd/change_plan/cmd.go`
   - Define a CLI command similar to the existing ones, e.g.:
     ```go
     package changeplan

     import (
         "fmt"
         "github.com/spf13/cobra"
     )

     func Command() *cobra.Command {
         cmd := &cobra.Command{
             Use:   "changeplan",
             Short: "Manage change plans",
         }
         cmd.AddCommand(addChangePlanCmd())
         return cmd
     }
     ```

5. **Implement Additional Logic to Handle Change Plans**:
   - Include validation to ensure proper data handling.
   - Implement necessary functions for CRUD operations on change plans.
   - Update logging to include information about change plan actions.

6. **Testing**:
   - Write tests for the newly created command and functionalities to ensure it behaves as expected.
   - File to Modify: `internal/load/load_test.go`. Create tests for the new change plan functions.

7. **Documentation**:
   - Update the `README.md` to include instructions on how to use the new feature, with examples. Indicate how it integrates with existing commands.
   - Example documentation for a command can be:
     ```markdown
     ### Change Plan Management
     - `aicoder changeplan add`: Add a new change plan.
     - `aicoder changeplan update <ID>`: Update an existing change plan.
     - `aicoder changeplan delete <ID>`: Delete a change plan.
     ```

8. **Review and Refine**:
   - Conduct a code review to ensure quality and adherence to guidelines before final merge.
   - Gather feedback from initial users after implementation to refine the feature based on their usage experience.

### Outcome
The implementation of change planning features will make AICoder a more robust tool for users wanting to manage their development workflows based on AI-powered insights.

AICoder is a AI-powered CLI that helps you code quickly.

1. load
1. search
1. plan
1. generate
1. write
1. check

## Why is AICoder necessary?

- **Fast and secure**: AICoder works in your local, LLM (e.g. OpenAI) is the only external endpoint that AICoder interacts with.
- **CI support**: you can use the same CLI in you CI. (PR review with domain knowledge of the repository.)
- **Compliance**: Not like [devlo.ai](https://devlo.ai/) or [devin.ai](https://devin.ai/) (which I love using), no need of installation and setup at organization level, which is not easy and quick for an enterprise company.
- **Personal**: the concept is to help you improve your productivity by accumulating the domain knowledge in your local and speed up your development speed.

## Environment Variables

- `OPENAI_API_KEY`

## PGVector

```
brew install postgresql@15
```

```sql
CREATE DATABASE aicoder;
CREATE EXTENSION IF NOT EXISTS vector;
CREATE USER aicoder WITH PASSWORD 'aicoder';
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO aicoder;
GRANT ALL ON SCHEMA public TO aicoder;
```

https://github.com/pgvector/pgvector-go

## Configuration

```yaml
repository: aicoder
load:
  exclude:
    - ent
  include:
    - ent/schema

search:
  top_n: 5
```