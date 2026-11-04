Output docs/assets/overview-demo.mp4
Source scripts/demo-settings.tape

# Preparing Sunday's order, with fixed mock data. Target: under 75 seconds.
Type "# Prepare Sunday's service. Mock data; your IDs will differ."
Enter
Sleep 2s

Type "# 1. Find the Sunday service type."
Enter
Type "pccli services service-types list"
Sleep 1s
Enter
Sleep 7s

Ctrl+L
Type "# 2. Choose its plan by date and title."
Enter
Type "pccli services plans list --service-type 3001"
Sleep 1s
Enter
Sleep 8s

Ctrl+L
Type "# 3. Review the order: Amazing Grace is song 1001."
Enter
Type "pccli services items list --service-type 3001 --plan 4001"
Sleep 1s
Enter
Sleep 10s
Screenshot docs/assets/overview-demo-poster.png

Ctrl+L
Type "# 4. Check the song's arrangements before rehearsal."
Enter
Type "pccli services songs show 1001"
Sleep 1s
Enter
# Final useful result remains visible; the player does not loop.
Sleep 12s
