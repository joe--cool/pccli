Output docs/assets/library-demo.gif
Set FontSize 16
Set Width 900
Set Height 760
Set Padding 24
Set TypingSpeed 55ms
Env PCCLI_MOCK "true"
Env PCCLI_COLOR "always"

Type "pccli"
Enter
Sleep 3s
Type "pccli services --help"
Enter
Sleep 2s
Type "pccli services songs search Amazing"
Enter
Sleep 2s
Type "pccli services songs show 'Amazing Grace'"
Enter
Sleep 2s
Type "pccli services songs arrangements 'Amazing Grace'"
Enter
Sleep 3s
Type "pccli services songs keys 'Amazing Grace'"
Enter
Sleep 2s
Type "pccli services songs attachments 'Amazing Grace' --arrangement 'Full Band'"
Enter
Sleep 2s
Type "pccli services songs attach 'Amazing Grace' --url https://example.com/amazing-grace-chart.pdf --filename amazing-grace-chart.pdf"
Enter
Sleep 3s
