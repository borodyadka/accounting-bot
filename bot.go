package accounting_bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

const VERSION = 1

type Config struct {
	AuthCode     string
	AdminContact string
}

type Bot struct {
	logger  *logrus.Logger
	token   string
	api     *tgbotapi.BotAPI
	storage Storage
	config  Config
	stopC   chan struct{}
	doneC   chan struct{}
}

func (b *Bot) handleError(chatID int64, err error) error {
	if err, ok := err.(fmt.Stringer); ok {
		_, _ = b.api.Send(tgbotapi.NewMessage(chatID, err.String())) // TODO: i18n
		return nil
	}
	if _, ok := err.(*UnknownCommandError); !ok {
		_, _ = b.api.Send(tgbotapi.NewMessage(chatID, "sorry, internal error :(")) // TODO: i18n
		return err
	}
	return nil
}

func (b *Bot) handle(update *tgbotapi.Update) error {
	var msg *tgbotapi.Message
	updated := false
	if update.Message != nil {
		msg = update.Message
	} else if update.EditedMessage != nil {
		msg = update.EditedMessage
		updated = true
	} else {
		return nil
	}
	if msg.Chat == nil {
		return nil
	}
	b.logger.WithFields(logrus.Fields{
		"id":   msg.MessageID,
		"text": msg.Text,
	}).Debug("handle message")

	cmd, err := ParseCommand(msg)
	if err != nil {
		return b.handleError(msg.Chat.ID, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	user, err := b.storage.GetUserByTelegramID(ctx, msg.Chat.ID)
	if err != nil {
		return b.handleError(msg.Chat.ID, err)
	}
	// TODO: check bot version and send changelog to user

	switch cmd.(type) {
	case *HelpCommand:
		_, _ = b.api.Send(Markdown(tgbotapi.NewMessage(msg.Chat.ID, manual)))
		return nil
	case *StartCommand:
		if user == nil {
			cmd := cmd.(*StartCommand)
			if b.config.AuthCode != "" && cmd.Code != b.config.AuthCode {
				return b.handleError(msg.Chat.ID, &InvalidAuthCodeError{})
			}
			user, err = b.storage.SaveUser(ctx, &User{
				TelegramID: msg.Chat.ID,
				Enabled:    true,
				Currency:   "USD",
				Features:   Features{},
			})
			if err != nil {
				return b.handleError(msg.Chat.ID, err)
			}
			_, _ = b.api.Send(
				// TODO: i18n
				Markdown(tgbotapi.NewMessage(
					msg.Chat.ID,
					fmt.Sprintf("Welcome aboard! Selected currency is %s\nTo change send `/currency RUB`", user.Currency),
				)),
			)
		}
		return nil
	}

	if user == nil || !user.Enabled {
		return b.handleError(msg.Chat.ID, &UserNotFoundError{})
	}
	switch cmd.(type) {
	case *CurrencyCommand:
		user.Currency = cmd.(*CurrencyCommand).Currency
		_, err := b.storage.SaveUser(ctx, user)
		if err != nil {
			return b.handleError(msg.Chat.ID, err)
		}
		_, _ = b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Saved")) // TODO: i18n
	case *DumpCommand:
		// TODO
	case *EntryCommand:
		entry, err := b.storage.SaveEntry(ctx, user, &cmd.(*EntryCommand).Entry)
		if err != nil {
			return b.handleError(msg.Chat.ID, err)
		}
		if !updated {
			// TODO: i18n
			addedMsg, err := b.api.Send(tgbotapi.NewMessage(
				msg.Chat.ID,
				fmt.Sprintf("Added %.2f%s", entry.Value, entry.Currency)),
			)
			if err != nil {
				return b.handleError(msg.Chat.ID, err)
			}
			entry.ReplyID = int64(addedMsg.MessageID)
			if err = b.storage.SaveReplyID(ctx, user, entry.MessageID, int64(addedMsg.MessageID)); err != nil {
				return b.handleError(msg.Chat.ID, err)
			}
		} else {
			// TODO: i18n
			_, err := b.api.Send(tgbotapi.NewEditMessageText(
				msg.Chat.ID,
				int(entry.ReplyID),
				fmt.Sprintf("Added %.2f%s", entry.Value, entry.Currency),
			))
			if err != nil && !strings.Contains(
				// we should not send error back in thisspecial case
				// only way to handle this error is compare text description
				err.Error(),
				"specified new message content and reply markup are exactly the same as a current content and reply markup of the message",
			) {
				return b.handleError(msg.Chat.ID, err)
			}
		}
	case *AddTagCommand:
		cmd := cmd.(*AddTagCommand)
		if err := b.storage.AddTag(ctx, user, cmd.SearchTag, cmd.Tags); err != nil {
			return b.handleError(msg.Chat.ID, err)
		}
		_, _ = b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Tags added")) // TODO: i18n
	case *RemoveTagCommand:
		cmd := cmd.(*RemoveTagCommand)
		if err := b.storage.RemoveTag(ctx, user, cmd.Tags); err != nil {
			return b.handleError(msg.Chat.ID, err)
		}
		_, _ = b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Tags removed")) // TODO: i18n
	case *ListTagsCommand:
		cmd := cmd.(*ListTagsCommand)
		tags, err := b.storage.ListTag(ctx, user, cmd.SearchTags)
		if err != nil {
			return b.handleError(msg.Chat.ID, err)
		}
		_, _ = b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Tags:\n"+strings.Join(tags, "\n"))) // TODO: i18n
	}

	return nil
}

func (b *Bot) Start() error {
	defer func() {
		b.doneC <- struct{}{}
	}()

	bot, err := tgbotapi.NewBotAPI(b.token)
	if err != nil {
		return err
	}
	b.api = bot
	b.logger.WithField("name", bot.Self.UserName).Info("started")

	updates, err := b.api.GetUpdatesChan(tgbotapi.UpdateConfig{Offset: 0, Limit: 0, Timeout: 60})
	if err != nil {
		return err
	}

	for {
		select {
		case update := <-updates:
			if err := b.handle(&update); err != nil {
				b.logger.WithError(err).Error("failed to handle message")
			}
		case <-b.stopC:
			return nil
		}
	}
}

func (b *Bot) Stop() error {
	b.stopC <- struct{}{}
	if b.api != nil {
		b.api.StopReceivingUpdates()
		b.api = nil
	}
	<-b.doneC
	b.logger.Info("stopped")
	return nil
}

func New(token string, logger *logrus.Logger, storage Storage, config Config) (*Bot, error) {
	return &Bot{
		logger:  logger,
		token:   token,
		storage: storage,
		config:  config,
		stopC:   make(chan struct{}, 1),
		doneC:   make(chan struct{}, 1),
	}, nil
}
