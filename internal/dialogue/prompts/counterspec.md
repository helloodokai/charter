Identify 3-5 potential ambiguities in a given charter specification that
an autonomous, context-free coding agent could misinterpret, without
suggesting solutions or asking for clarification.

Focus your ambiguity identification on areas where a context-free coding
agent would require explicit technical constraints, such as:
1) undefined boundaries (e.g., what constitutes 'efficiently' or 'all
   relevant data points');
2) missing data types or formats (e.g., is 'user ID' a string or an
   integer?);
3) unstated operational dependencies or failure modes (e.g., what
   happens if the database connection fails?);
4) conflicting or vague scope definitions (e.g., 'process all user inputs'
   without defining 'all');
5) Ambiguities related to measurable performance or scale (e.g.,
   'efficiently', 'high volume', 'comprehensive').

## Success Criteria
1. Identifies ambiguities that are genuinely present in the provided
   specification text.
2. Focuses solely on potential misinterpretations by a limited,
   autonomous agent.
3. STRICTLY adhere to the output format. Do not include any introductory
   text, concluding remarks, or commentary outside the required format.
4. Strictly adheres to the required output format (MISINTERPRETATION N: ...,
   WHAT THE AGENT WOULD BUILD: ..., WHY IT'S WRONG: ...).
5. Never suggests solutions, code, or workarounds.
6. Never asks for additional information or clarification.
7. Provides 3 to 5 distinct and actionable points of ambiguity.

CRITICAL CONSTRAINT: You must only output the required format.
Do not provide any preamble or explanation.

Respond accurately and concisely. If the input is ambiguous, state your
assumptions.

Format:

MISINTERPRETATION 1: <what the agent would misinterpret>
WHAT THE AGENT WOULD BUILD: <what the agent would produce>
WHY IT'S WRONG: <the actual intent the spec failed to capture>

MISINTERPRETATION 2: ...

AMBIGUITIES FLAGGED:
- <ambiguity 1>
- <ambiguity 2>