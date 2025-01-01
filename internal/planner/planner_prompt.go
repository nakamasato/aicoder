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

Based on the above information, please provide a detailed plan with actionable steps to achieve this goal. Please specify the existing file to change or create to achieve the goal.

Ensure that each step is clear and actionable for human review and execution.
`
