Output docs/assets/screencasts/ops-execute-smoke-test.gif

Require zsh

Set Shell "zsh"
Set FontFamily "Andale Mono"
Set FontSize 16
Set Width 1600
Set Height 640
Set Padding 16
Set WindowBar Colorful
Set TypingSpeed 35 ms
Set WaitTimeout 30m
Set Theme "TokyoNight"

Env HOME "/tmp/c8volt-vhs/home"
Env XDG_CONFIG_HOME "/tmp/c8volt-vhs/home/.config"

Hide
Type "cd /tmp/c8volt-vhs/bin"
Enter
Type "mkdir -p /tmp/c8volt-vhs/reports"
Enter
Type "clear"
Enter
Show

Wait
Sleep 2s
Hide
Type "printf '\\033[1A\\033[2K%s\\n' 'Preview the smoke-test workflow before it touches the cluster.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt ops execute smoke-test --dry-run"
Enter
Wait

Sleep 1s
Hide
Type "printf '\\033[1A\\033[2K\\n%s\\n' 'Run the smoke test and write an audit report.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt ops execute smoke-test --auto-confirm --report-file /tmp/c8volt-vhs/reports/smoke-test.md"
Enter
Wait

Sleep 1s
Hide
Type "printf '\\033[1A\\033[2K\\n%s\\n' 'Open the first part of the report.'"
Enter
Show
Sleep 1s
Wait
Type "sed -n '1,80p' /tmp/c8volt-vhs/reports/smoke-test.md"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Execute smoke test demo complete.'"
Enter
Show
Wait
Sleep 3s
