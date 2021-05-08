package accounting_bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

func Markdown(msg tgbotapi.MessageConfig) tgbotapi.MessageConfig {
	msg.ParseMode = "markdown"
	return msg
}
