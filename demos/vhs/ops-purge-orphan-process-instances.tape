Output docs/assets/screencasts/ops-purge-orphan-process-instances.gif

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
Type "printf '\033[1A\033[2K%s\n' 'Preview orphan process-instance purge before deleting anything.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt ops purge orphan-process-instances --dry-run"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Run orphan process-instance purge and write an audit report.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt ops purge orphan-process-instances --report-file /tmp/c8volt-vhs/reports/orphan-purge.md"
Enter
Sleep 1s
Type "y"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Open the first part of the report.'"
Enter
Show
Sleep 1s
Wait
Type "head -n 20 /tmp/c8volt-vhs/reports/orphan-purge.md"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Purge orphan process instances demo complete.'"
Enter
Show
Wait
Sleep 3s
