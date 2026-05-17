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
Type "echo 'Check the connection and credentials.'"
Enter
Wait
Type "./c8volt config test-connection"
Enter
Wait

Sleep 1s
Type "echo 'List deployed process definitions.'"
Enter
Wait
Type "./c8volt get pd"
Enter
Wait

Sleep 1s
Type "echo 'Show bundled BPMN resources available for deployment.'"
Enter
Wait
Type "./c8volt embed list"
Enter
Wait

Sleep 1s
Type "echo 'Deploy all bundled demo process definitions for configured Camunda version only.'"
Enter
Wait
Type "./c8volt embed deploy --all"
Enter
Wait

Sleep 1s
Type "echo 'Show deployed process definitions with their instance counts.'"
Enter
Wait
Type "./c8volt get pd --stat"
Enter
Wait

Sleep 1s
Type "echo 'Count process instances for the demo process.'"
Enter
Wait
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Sleep 1s
Type "echo 'Start one demo process instance with process variables.'"
Enter
Wait
Type `./c8volt run pi -b C89_MultipleSubProcessesParentProcess --vars '{"businessId":123}'`
Enter
Wait

Sleep 1s
Type "echo 'Confirm the new process instance is visible.'"
Enter
Wait
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Sleep 1s
Type "echo 'Refresh process definition stats after starting the instance.'"
Enter
Wait
Type "./c8volt get pd --stat"
Enter
Wait

Sleep 1s
Type "echo 'Inspect process instances with their variables.'"
Enter
Wait
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --with-vars"
Enter
Wait

Sleep 1s
Type "echo 'Store one process instance key for follow-up commands.'"
Enter
Wait
Type "C8_VOLT_PI=$(./c8volt get pi -b C89_MultipleSubProcessesParentProcess --limit 1 --keys-only)"
Enter
Wait

Sleep 1s
Type "echo 'Show the selected process instance key.'"
Enter
Wait
Type "echo $C8_VOLT_PI"
Enter
Wait

Sleep 1s
Type "echo 'Walk the selected process instance and its related executions.'"
Enter
Wait
Type "./c8volt walk pi --key $C8_VOLT_PI"
Enter
Wait

Sleep 1s
Type "echo 'Preview which demo process instances would be deleted.'"
Enter
Wait
Type "./c8volt delete pi -b C89_MultipleSubProcessesParentProcess --force --dry-run"
Enter
Sleep 1s
Type "y"
Enter
Wait

Sleep 1s
Type "echo 'Delete the demo process instances to clean up.'"
Enter
Wait
Type "./c8volt delete pi -b C89_MultipleSubProcessesParentProcess --force"
Enter
Sleep 1s
Type "y"
Enter
Wait

Sleep 1s
Type "echo 'Verify no demo process instances remain.'"
Enter
Wait
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Type "echo 'Fast start demo complete.'"
Enter
Wait
