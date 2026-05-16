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
Set Theme "TokyoNight"

Env HOME "/tmp/c8volt-vhs/home"
Env XDG_CONFIG_HOME "/tmp/c8volt-vhs/home/.config"

Hide
Type "cd /tmp/c8volt-vhs/bin"
Enter
Type "clear"
Enter
Show

Sleep 500 ms
Type "./c8volt config test-connection"
Enter
Sleep 3s

Type "./c8volt get pd"
Enter
Sleep 3s

Type "./c8volt embed list"
Enter
Sleep 3s

Type "./c8volt embed deploy --all"
Enter
Sleep 2s

Type "./c8volt get pd --stat"
Enter
Sleep 2s

Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Sleep 2s

Type `./c8volt run pi -b C89_MultipleSubProcessesParentProcess --vars '{"businessId":123}'`
Enter
Sleep 4s

Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Sleep 2s

Type "./c8volt get pd --stat"
Enter
Sleep 2s

Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --with-vars"
Enter
Sleep 5s

Type "C8_VOLT_PI=$(./c8volt get pi -b C89_MultipleSubProcessesParentProcess --limit 1 --keys-only)"
Enter
Sleep 2s

Type "echo $C8_VOLT_PI"
Enter
Sleep 1s

Type "./c8volt walk pi --key $C8_VOLT_PI"
Enter
Sleep 3s

Type "./c8volt delete pi -b C89_MultipleSubProcessesParentProcess --force"
Enter
Sleep 1s
Type "y"
Enter
Sleep 33s

Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Sleep 2s
