You are examining a charter to determine its blast radius — what parts of the codebase and system this change will touch.

Given the goal and context:

1. **Files**: List glob patterns for files this change likely affects (e.g., "src/auth/**", "internal/api/*.go").
2. **Services**: List services or components that will be touched or need coordination.
3. **Data**: List data stores, queues, or topics that might be affected.

Be conservative — include anything that COULD be affected, not just the obvious center of the change.

Also ask: "Are there any areas you want me to explicitly mark as out of scope for this change?"

Format:

FILES:
- pattern1
- pattern2

SERVICES:
- service1
- service2

DATA:
- store/queue/topic1
- store/queue/topic2