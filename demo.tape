Output demo.gif
Set FontSize 14
Set Width 1200
Set Height 800
Set PlaybackSpeed 1.5
Set Theme "Catppuccin Mocha"

Type "cd /tmp/demo-repo && export PATH=/Users/lewis/Code/labs/charter/dist:/Users/lewis/Code/labs/acig/dist:$PATH"
Enter
Sleep 300ms

Type ""
Enter
Sleep 100ms

Type "echo '  ╔═══════════════════════════════════════════════╗'"
Enter
Type "echo '  ║     Charter — Spec-driven Code Review         ║'"
Enter
Type "echo '  ╚═══════════════════════════════════════════════╝'"
Enter
Sleep 1s

Type ""
Enter
Sleep 200ms

Type "echo '📋 Step 1: Validate a charter'"
Enter
Type "charter validate .charters/ch-2026-05-05-rate010.yaml"
Enter
Sleep 1s

Type ""
Enter
Sleep 200ms

Type "echo '📋 Step 2: List charters in this repo'"
Enter
Type "charter ls"
Enter
Sleep 1s

Type ""
Enter
Sleep 200ms

Type "echo '✅ Step 3: Conformance — conforming diff'"
Enter
Type "echo '  Diff stays within blast radius: src/api/**, internal/middleware/**'"
Enter
Sleep 300ms
Type "git diff main..feat/rate-limit | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format md"
Enter
Sleep 2s

Type ""
Enter
Sleep 200ms

Type "echo '⚠️  Step 4: Conformance — NON-conforming diff'"
Enter
Type "echo '  Diff touches src/auth/ — outside declared blast radius!'"
Enter
Sleep 300ms
Type "cat bad-diff.patch | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format md"
Enter
Sleep 2s

Type ""
Enter
Sleep 200ms

Type "echo '📊 Step 5: JSON output for ACIG consumption'"
Enter
Type "cat bad-diff.patch | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format json"
Enter
Sleep 2s

Type ""
Enter
Sleep 200ms

Type "echo '🔒 Step 6: Approve the charter'"
Enter
Type "charter approve .charters/ch-2026-05-05-rate010.yaml"
Enter
Sleep 1s

Type ""
Enter
Sleep 200ms

Type "echo '📐 Step 7: Charter JSON Schema'"
Enter
Type "charter schema | jq '.properties | keys'"
Enter
Sleep 1s

Type ""
Enter
Sleep 200ms

Type "echo ''"
Enter
Type "echo 'Full pipeline:'"
Enter
Type "echo '  Issue ──► Charter ──► charter.yaml ──► Agent ──► PR ──► ACIG'"
Enter
Type "echo '           (harden)    (contract)       (code)    (diff)  (verify)'"
Enter
Sleep 2s