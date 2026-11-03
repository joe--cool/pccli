Output docs/assets/attachment-demo.mp4
Source scripts/demo-settings.tape

Type "# Put files in the right place"
Enter
Type "# Mock data. Same song; three different attachment destinations."
Enter
Sleep 3s

Type "# 1. Song-wide files: useful across arrangements."
Enter
Type "pccli services attachments list --song 1001"
Sleep 1s
Enter
Sleep 8s
Screenshot docs/assets/attachment-demo-poster.png

Ctrl+L
Type "# 2. Full Band files: specific to arrangement 2001."
Enter
Type "pccli services attachments list --song 1001 --arrangement 2001"
Sleep 1s
Enter
Sleep 8s

Ctrl+L
Type "# 3. Key-specific files: Full Band, key record 3001."
Enter
Type "pccli services attachments list --song 1001 --arrangement 2001 --key 3001"
Sleep 1s
Enter
Sleep 8s
Type "# Use the same parent flags when adding or updating a file."
Enter
Sleep 10s
