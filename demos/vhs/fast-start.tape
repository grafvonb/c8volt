Output docs/assets/screencasts/fast-start.gif
Output docs/assets/screencasts/fast-start.mp4

Require zsh

Set Shell "zsh"
Set FontFamily "Andale Mono"
Set FontSize 16
Set Width 1440
Set Height 760
Set Padding 18
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
Type "./c8volt version"
Enter
Sleep 2s

Type "./c8volt config validate"
Enter
Sleep 2s

Type "./c8volt config test-connection"
Enter
Sleep 3s

Type "./c8volt get cluster version"
Enter
Sleep 2s

Type "./c8volt get cluster topology"
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
Sleep 5s

Type "./c8volt get pd --stat"
Enter
Sleep 3s

Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Sleep 2s

Type `./c8volt run pi -b C89_MultipleSubProcessesParentProcess --vars '{"businessId":123}'`
Enter
Sleep 5s

Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Sleep 2s

Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --with-vars"
Enter
Sleep 5s

Type "./c8volt delete pi -b C89_MultipleSubProcessesParentProcess --force"
Enter
Sleep 1s
Type "y"
Enter
Sleep 35s

Type "./c8volt get pi -b C89_MultipleSubProcessesParentProcess --total"
Enter
Sleep 2s
