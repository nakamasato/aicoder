package planner

const PLANNER_PROMPT = `You are a helpful assistant that generates detailed action plans based on provided project information.
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
`


const REPLAN_PROMPT = `You are a helpful assistant that generates detailed action plans based on provided project information.
The plan you've just made failed the validation.
Please provide a new plan based on the provided feedback.

-----------------------
Goal: %s
-----------------------
Possibly relevant documents:
%s
-----------------------
Previous plan:
%s
-----------------------
Previous errors:
%s
-----------------------
`
