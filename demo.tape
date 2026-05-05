Output demo.gif
Set FontSize 16
Set Width 1200
Set Height 800
Set PlaybackSpeed 0.8
Set Theme "Catppuccin Mocha"

Type "cd /tmp/demo-repo && export PATH=/Users/lewis/Code/labs/charter/dist:/Users/lewis/Code/labs/acig/dist:$PATH"
Enter
Sleep 500ms

Type ""
Enter
Sleep 300ms

Type "echo '  ╔═══════════════════════════════════════════════╗'"
Enter
Sleep 100ms
Type "echo '  ║     Charter — Spec-driven Code Review         ║'"
Enter
Sleep 100ms
Type "echo '  ╚═══════════════════════════════════════════════╝'"
Enter
Sleep 3s

Type ""
Enter
Sleep 500ms

Type "echo '📋 Step 1: Validate a charter'"
Enter
Sleep 1s
Type "charter validate .charters/ch-2026-05-05-rate010.yaml"
Enter
Sleep 3s

Type ""
Enter
Sleep 2s

Type "echo '📋 Step 2: List charters in this repo'"
Enter
Sleep 1s
Type "charter ls"
Enter
Sleep 3s

Type ""
Enter
Sleep 2s

Type "echo '✅ Step 3: Conformance — conforming diff'"
Enter
Sleep 500ms
Type "echo '  Diff stays within blast radius: src/api/**, internal/middleware/**'"
Enter
Sleep 2s
Type "git diff main..feat/rate-limit | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format md"
Enter
Sleep 4s

Type ""
Enter
Sleep 2s

Type "echo '⚠️  Step 4: Conformance — NON-conforming diff'"
Enter
Sleep 500ms
Type "echo '  Diff touches src/auth/ — outside declared blast radius!'"
Enter
Sleep 2s
Type "cat bad-diff.patch | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format md"
Enter
Sleep 4s

Type ""
Enter
Sleep 2s

Type "echo '📊 Step 5: JSON output for ACIG consumption'"
Enter
Sleep 500ms
Type "cat bad-diff.patch | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format json"
Enter
Sleep 4s

Type ""
Enter
Sleep 2s

Type "echo '🔒 Step 6: Approve the charter'"
Enter
Sleep 500ms
Type "charter approve .charters/ch-2026-05-05-rate010.yaml"
Enter
Sleep 3s

Type ""
Enter
Sleep 2s

Type "echo '📐 Step 7: Charter JSON Schema'"
Enter
Sleep 500ms
Type "charter schema | jq '.properties | keys'"
Enter
Sleep 3s

Type ""
Enter
Sleep 2s

Type "echo ''"
Enter
Sleep 300ms
Type "echo 'Full pipeline:'"
Enter
Sleep 300ms
Type "echo '  Issue ──► Charter ──► charter.yaml ──► Agent ──► PR ──► ACIG'"
Enter
Sleep 300ms
Type "echo '           (harden)    (contract)       (code)    (diff)  (verify)'"
Enter
Sleep 4s