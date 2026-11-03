Output docs/assets/overview-demo.mp4
Source scripts/demo-settings.tape

Type "# pccli: explore your Planning Center song library"
Enter
Type "# Mock data. Find a song, inspect it, then choose a key."
Enter
Sleep 3s

Type "# 1. Find a song by title; keep its ID for the next step."
Enter
Type 'pccli services songs list --search "Amazing"'
Sleep 1s
Enter
Sleep 7s

# Clear only after the result has had time to be read.
Ctrl+L
Type "# 2. Inspect the song and compare its arrangements."
Enter
Type "pccli services songs show 1001"
Sleep 1s
Enter
Sleep 12s
Screenshot docs/assets/overview-demo-poster.png

Ctrl+L
Type "# 3. Choose Full Band (2001) and see its available keys."
Enter
Type "pccli services keys list --song 1001 --arrangement 2001"
Sleep 1s
Enter
Sleep 7s
Type "# Next: use these IDs to maintain metadata, charts, and files."
Enter
# Leave the result visible. The website player stops here; it does not loop.
Sleep 10s
