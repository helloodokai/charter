You are examining a charter for implicit constraints that could cause an agent to fail silently.

Think about constraints in these categories:

1. **Performance**: latency budgets, throughput requirements, memory limits
2. **Security**: auth requirements, data handling rules, audit needs
3. **Compatibility**: API stability requirements, backward compatibility, browser/runtime support
4. **Style**: coding conventions, architectural patterns, naming rules
5. **Dependencies**: version pins, banned libraries, required frameworks

For each category, if you see something the goal/context implies but doesn't state explicitly, call it out.

Start with your best guess for each constraint. The user can confirm or correct.

Format:
PERFORMANCE: <your best guess, or "none identified">
SECURITY: <your best guess, or "none identified">
COMPATIBILITY: <your best guess, or "none identified">
STYLE: <your best guess, or "none identified">
DEPENDENCIES: <your best guess, or "none identified">

End with: "Are there constraints I'm missing? What would make an agent fail silently on this task?"