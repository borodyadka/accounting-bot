package accounting_bot

import "strings"

var manual = `
/help — show this help
/dump <format> <period>
<amount> <comment with tags>
`

func init() {
	manual = strings.TrimSpace(manual)
}
