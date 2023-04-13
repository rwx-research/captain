package templating

import (
	"regexp"
	"strings"
)

func ShellEscape(value string) string {
	return strings.ReplaceAll(value, "'", `'"'"'`)
}

func RegexpEscape(value string) string {
	return regexp.QuoteMeta(value)
}
