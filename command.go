package accounting_bot

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/text/currency"
)

var (
	reHelp = regexp.MustCompile(`^/help`)
	reDump = regexp.MustCompile(`^/(?P<cmd>dump\s*)`)
	reStat = regexp.MustCompile(`^/(?P<cmd>(sum|max|maximum|min|minimum|avg|average|med|median)\B)`) // TODO
	// /start [code] - to authorize user
	reStart = regexp.MustCompile(`^/(?P<cmd>start)(\s+(?P<code>[\w\d]+))?`)
	// /currency RUB - to set user currency
	reCurrency = regexp.MustCompile(`^/(?P<cmd>currency)(\s+(?P<code>[\w]{3}))`)
	// /tag #burger #food - to add tag #food to all #burger entries
	reTag = regexp.MustCompile(`^/tag\s+`)
	// /untag #burger - to remove all #burger tags (not entries)
	reUntag = regexp.MustCompile(`^/untag\s+`)
	// /tags - list all tags and number of usages
	// /tags #food - list all tags on entries with #food tag
	reTags = regexp.MustCompile(`^/tags\s*`)
	// <value> [comment with #hashtags]
	reEntry      = regexp.MustCompile(`^(?P<value>\d+(\.\d+)?)(?P<comment>\s?.*)$`)
	reHashTags   = regexp.MustCompile(`(\B#[\p{L}\d]+)`)
	rePeriod     = regexp.MustCompile(`(((?P<period>\d+)\s+)?(?P<modifier>years?|months?|weeks?|days?|hours?))`)
	reDumpFormat = regexp.MustCompile(`(?P<format>csv|sqlite)`)
)

type Command interface{}

type HelpCommand struct{}

type StartCommand struct {
	Code string
}

type CurrencyCommand struct {
	Currency string
}

type DumpCommand struct {
	From   time.Time
	Format string
	Tags   []string
}

type StatCommand struct {
	From time.Time
	Tags []string
}

type EntryCommand struct {
	Entry Entry
}

type AddTagCommand struct {
	SearchTag string
	Tags      []string
}

type RemoveTagCommand struct {
	Tags []string
}

type ListTagsCommand struct {
	SearchTags []string
}

func extractHashTags(s string) []string {
	tags := reHashTags.FindAllString(s, -1)
	if len(tags) == 0 {
		tags = make([]string, 0)
	}
	return tags
}

func getPeriodBeginning(period int, modifier string) time.Time {
	now := time.Now()
	result := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	switch modifier {
	case "year", "years":
		result = result.AddDate(-period, 0, 0)
	case "month", "months":
		result = result.AddDate(0, -period, 0)
	case "week", "weeks":
		result = result.AddDate(0, 0, -period*7)
	case "day", "days":
		result = result.AddDate(0, 0, -period)
	case "hour", "hours":
		result = result.Add(-time.Duration(period) * time.Hour)
	}

	return result
}

func getDumpFormat(s string) string {
	matches := reDumpFormat.FindAllString(s, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func ParseCommand(message *tgbotapi.Message) (Command, error) {
	s := strings.TrimSpace(message.Text)
	// show help
	if reHelp.Match([]byte(s)) {
		return &HelpCommand{}, nil
	}
	// start bot for user
	if m, ok := getMatches(reStart, s); ok {
		return &StartCommand{Code: m["code"]}, nil
	}
	// request dump
	if reDump.Match([]byte(s)) {
		cmd := &DumpCommand{
			From:   time.Time{},
			Format: "csv",
			Tags:   extractHashTags(s),
		}
		if mf, ok := getMatches(reDumpFormat, s); ok {
			cmd.Format = mf["format"]
		}
		if mp, ok := getMatches(rePeriod, s); ok {
			sp := strings.TrimSpace(mp["period"])
			var period int64 = 1
			if sp != "" {
				var err error
				period, err = strconv.ParseInt(sp, 10, 32)
				if err != nil {
					return nil, &InvalidSyntaxError{ /*TODO: more info*/ }
				}
			}
			cmd.From = getPeriodBeginning(int(period), mp["modifier"])
		}
		return cmd, nil
	}
	// add entry
	if m, ok := getMatches(reEntry, s); ok {
		value, err := strconv.ParseFloat(m["value"], 64)
		if err != nil {
			return nil, NewInternalError(err)
		}
		hashtags := extractHashTags(m["comment"])
		return &EntryCommand{
			Entry{
				CreatedAt: time.Now(),
				Comment:   strings.TrimSpace(m["comment"]),
				Tags:      hashtags,
				Value:     float32(value),
				MessageID: int64(message.MessageID),
			},
		}, nil
	}
	// set currency
	if m, ok := getMatches(reCurrency, s); ok {
		_, err := currency.ParseISO(m["code"])
		if err != nil {
			return nil, &InvalidCurrencyError{Currency: m["code"]}
		}
		return &CurrencyCommand{Currency: m["code"]}, nil
	}

	if reTag.Match([]byte(s)) {
		hashtags := extractHashTags(s)
		if len(hashtags) < 2 {
			return nil, &InvalidSyntaxError{ /*TODO: more info*/ }
		}
		return &AddTagCommand{
			SearchTag: hashtags[0],
			Tags:      hashtags[1:],
		}, nil
	}
	if reUntag.Match([]byte(s)) {
		hashtags := extractHashTags(s)
		if len(hashtags) < 1 {
			return nil, &InvalidSyntaxError{ /*TODO: more info*/ }
		}
		return &RemoveTagCommand{
			Tags: hashtags,
		}, nil
	}
	if reTags.Match([]byte(s)) {
		return &ListTagsCommand{
			SearchTags: extractHashTags(s),
		}, nil
	}

	if reDump.Match([]byte(s)) {
		cmd := &DumpCommand{
			Format: getDumpFormat(s),
			Tags:   extractHashTags(s),
		}
		if mp, ok := getMatches(rePeriod, s); ok {
			sp := strings.TrimSpace(mp["period"])
			var period int64 = 1
			if sp != "" {
				var err error
				period, err = strconv.ParseInt(sp, 10, 32)
				if err != nil {
					return nil, &InvalidSyntaxError{ /*TODO: more info*/ }
				}
			}
			cmd.From = getPeriodBeginning(int(period), mp["modifier"])
		}
		return cmd, nil
	}

	return nil, &UnknownCommandError{Command: s}
}
