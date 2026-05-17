Output docs/assets/screencasts/fast-start.gif

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
Type "clear"
Enter
Show

Wait
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Check the connection and credentials.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt config test-connection"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'List deployed process definitions.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt get pd"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Show bundled BPMN resources available for deployment.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt embed list"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Deploy all bundled demo process definitions for configured Camunda version only.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt embed deploy --all"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Show deployed process definitions with their instance counts.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt get pd --stat"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Count process instances for the demo process.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Start one demo process instance with process variables.'"
Enter
Show
Sleep 1s
Wait
Type `./c8volt run pi -b C89_MultipleSubProcessesParentProcess --vars '{"businessId":123}'`
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Confirm the new process instance is visible.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Refresh process definition stats after starting the instance.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt get pd --stat"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Inspect process instances with their variables.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --with-vars"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Store one process instance key for follow-up commands.'"
Enter
Show
Sleep 1s
Wait
Type "C8_VOLT_PI=$(./c8volt get pi -b C89_MultipleSubProcessesParentProcess --limit 1 --keys-only)"
Enter
Wait
Type "echo $C8_VOLT_PI"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Walk the selected process instance and its related executions.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt walk pi --key $C8_VOLT_PI"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Preview which demo process instances would be deleted.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt delete pi -b C89_MultipleSubProcessesParentProcess --force --dry-run"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Delete the demo process instances to clean up.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt delete pi -b C89_MultipleSubProcessesParentProcess --force"
Enter
Sleep 1s
Type "y"
Enter
Wait

Sleep 1s
Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Verify no demo process instances remain.'"
Enter
Show
Sleep 1s
Wait
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Hide
Type "printf '\033[1A\033[2K\n%s\n' 'Fast start demo complete.'"
Enter
Show
Wait
