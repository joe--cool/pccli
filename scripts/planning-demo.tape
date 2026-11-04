Output docs/assets/planning-demo.mp4
Source scripts/demo-settings.tape

# Check readiness for one chosen plan. Target: under 90 seconds.
Type "# Check Sunday's plan, times, teams, and responses. Mock data."
Enter
Sleep 2s

Type "# 1. Review the plan and unfilled-position count."
Enter
Type "pccli services plans show 4001 --service-type 3001"
Sleep 1s
Enter
Sleep 10s

Ctrl+L
Type "# 2. Confirm the service time."
Enter
Type "pccli services plan-times list --service-type 3001 --plan 4001"
Sleep 1s
Enter
Sleep 7s

Ctrl+L
Type "# 3. Check the service type's available teams."
Enter
Type "pccli services teams list --service-type 3001"
Sleep 1s
Enter
Sleep 7s

Ctrl+L
Type "# 4. Review assignments and who has responded."
Enter
Type "pccli services scheduled-people list --service-type 3001 --plan 4001"
Sleep 1s
Enter
Sleep 1s
Screenshot docs/assets/planning-demo-poster.png
# End on the response summary with time to read or pause.
Sleep 11s
