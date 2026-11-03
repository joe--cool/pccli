Output docs/assets/library-demo.gif
Set FontSize 16
Set Width 1000
Set Height 760
Set Padding 24
Set TypingSpeed 45ms
Env PCCLI_MOCK "true"
Env PCCLI_COLOR "always"

Type "pccli"
Enter
Sleep 2s
Type "pccli services songs list --search Amazing"
Enter
Sleep 2s
Type "pccli services songs show 1001"
Enter
Sleep 3s
Type "pccli services arrangements list --song 1001"
Enter
Sleep 2s
Type "pccli services keys list --song 1001 --arrangement 2001"
Enter
Sleep 2s
Type "pccli services attachments list --song 1001 --arrangement 2001"
Enter
Sleep 3s
