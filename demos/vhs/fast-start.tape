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
Type "./c8volt config test-connection"
Enter
Wait

Sleep 1s
Type "./c8volt get pd"
Enter
Wait

Sleep 1s
Type "./c8volt embed list"
Enter
Wait

Sleep 1s
Type "./c8volt embed deploy --all"
Enter
Wait

Sleep 1s
Type "./c8volt get pd --stat"
Enter
Wait

Sleep 1s
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Sleep 1s
Type `./c8volt run pi -b C89_MultipleSubProcessesParentProcess --vars '{"businessId":123}'`
Enter
Wait

Sleep 1s
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Sleep 1s
Type "./c8volt get pd --stat"
Enter
Wait

Sleep 1s
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --with-vars"
Enter
Wait

Sleep 1s
Type "C8_VOLT_PI=$(./c8volt get pi -b C89_MultipleSubProcessesParentProcess --limit 1 --keys-only)"
Enter
Wait

Sleep 1s
Type "echo $C8_VOLT_PI"
Enter
Wait

Sleep 1s
Type "./c8volt walk pi --key $C8_VOLT_PI"
Enter
Wait

Sleep 1s
Type "./c8volt delete pi -b C89_MultipleSubProcessesParentProcess --force"
Enter
Sleep 1s
Type "y"
Enter
Wait

Sleep 1s
Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Wait

Type "echo 'Fast start demo complete.'"
Enter
Wait
