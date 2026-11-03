Output docs/assets/json-demo.mp4
Source scripts/demo-settings.tape
Require jq

Type "# Use your song library in a script"
Enter
Type "# Mock data. jq selects fields from pccli's JSON envelope."
Enter
Sleep 3s

Type "set -o pipefail"
Sleep 1s
Enter
Sleep 2s

Type "# 1. Request all visible songs; inspect schema and completeness."
Enter
Type 'pccli --json services songs list --all | jq "{schema, meta}"'
Sleep 1s
Enter
Sleep 10s
Screenshot docs/assets/json-demo-poster.png

Ctrl+L
Type "# 2. Extract the IDs and titles your script needs."
Enter
Type "pccli --json services songs list --all | jq '.data[] | {id, title}'"
Sleep 1s
Enter
Sleep 8s
Type "# In scripts, check pccli's exit status before processing results."
Enter
Sleep 10s
