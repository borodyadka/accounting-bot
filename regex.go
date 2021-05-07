package accounting_bot

import (
	"regexp"
	"strings"
)

func getMatches(r *regexp.Regexp, str string) (map[string]string, bool) {
	if !r.Match([]byte(strings.TrimSpace(str))) {
		return nil, false
	}
	matches := r.FindStringSubmatch(strings.TrimSpace(str))
	result := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 {
			result[name] = matches[i]
		}
	}

	return result, true
}
