package planner

const PLANNER_GOAL_PROMPT = `You are a helpful assistant that generates detailed action plans based on provided project information.
-----------------------
Files in the repository:
%s
-----------------------
Possibly relevant documents:
%s

------------------------
My goal is: %s

Based on the above information, please provide a detailed plan with actionable steps to achieve this goal.
Please specify the existing file to change or create to achieve the goal.
For existing file, specify the line number (starting with 1) to change or delete.

Multiple changes cannot be made for the same file. If you need multiple changes on the file. please update the target lines by adding and deleteing the content in the target lines.
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
