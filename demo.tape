Output demo.gif
Set FontSize 16
Set Width 1200
Set Height 800
Set PlaybackSpeed 0.7
Set Theme "Catppuccin Mocha"

Type "cd /tmp/demo-repo"
Enter
Sleep 500ms

Type ""
Enter
Sleep 2s

Type "echo 'Step 1: Draft a charter — just describe what you want'"
Enter
Sleep 1s
Type "charter draft 'Add rate limiting to the public API' --non-interactive --turn-budget 3 --no-transcript"
Enter
Sleep 8s

Type ""
Enter
Sleep 2s

Type "ls .charters/"
Enter
Sleep 2s

Type ""
Enter
Sleep 2s

Type "echo 'The draft needs details — validate shows what to fill in:'"
Enter
Sleep 500ms

Type "echo 'Step 2: Validate the completed charter'"
Enter
Sleep 500ms
Type "echo 'After detailing non-goals, blast radius, constraints...'"
Enter
Sleep 1s
Type "charter validate .charters/ch-2026-05-05-rate010.yaml"
Enter
Sleep 3s

Type ""
Enter
Sleep 2s

Type "echo 'Step 3: List charters in this repo'"
Enter
Sleep 500ms
Type "charter ls"
Enter
Sleep 3s

Type ""
Enter
Sleep 2s

Type "echo 'Step 4: Conformance - diff stays within blast radius'"
Enter
Sleep 500ms
Type "git diff main..feat/rate-limit | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format md"
Enter
Sleep 4s

Type ""
Enter
Sleep 2s

Type "echo 'Step 5: Conformance - diff VIOLATES blast radius'"
Enter
Sleep 500ms
Type "echo '  Touching src/auth/ which is NOT in blast radius!'"
Enter
Sleep 2s
Type "cat bad-diff.patch | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format md"
Enter
Sleep 4s

Type ""
Enter
Sleep 2s

Type "echo 'Step 6: JSON output (what ACIG consumes)'"
Enter
Sleep 500ms
Type "cat bad-diff.patch | charter conformance .charters/ch-2026-05-05-rate010.yaml --diff - --format json"
Enter
Sleep 4s

Type ""
Enter
Sleep 2s

Type "echo 'Step 7: Approve the charter'"
Enter
Sleep 500ms
Type "charter approve .charters/ch-2026-05-05-rate010.yaml"
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
Type "echo '  Issue -> Charter -> charter.yaml -> Agent -> PR -> ACIG'"
Enter
Sleep 300ms
Type "echo '           (harden)   (contract)      (code)   (diff) (verify)'"
Enter
Sleep 5s