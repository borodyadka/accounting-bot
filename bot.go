package accounting_bot

import (
	"context"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"time"
)

const VERSION = 1

type Bot struct {
	logger  *logrus.Logger
	token   string
	api     *tgbotapi.BotAPI
	storage Storage
	stopC   chan struct{}
	doneC   chan struct{}
}

func (b *Bot) handleError(chatID int64, err error) error {
	if err, ok := err.(fmt.Stringer); ok {
		_, _ = b.api.Send(tgbotapi.NewMessage(chatID, err.String())) // TODO: i18n
	}
	if _, ok := err.(*UnknownCommandError); !ok {
		_, _ = b.api.Send(tgbotapi.NewMessage(chatID, "sorry, internal error :(")) // TODO: i18n
		return err
	}
	return nil
}

func (b *Bot) handle(update *tgbotapi.Update) error {
	var msg *tgbotapi.Message
	if update.Message != nil {
		msg = update.Message
	} else if update.EditedMessage != nil {
		msg = update.EditedMessage
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
		_, _ = b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, manual))
		return nil
	case *StartCommand:
		if user == nil {
			// TODO: check auth code
			_, err := b.storage.SaveUser(ctx, &User{
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
				tgbotapi.NewMessage(
					msg.Chat.ID,
					"Welcome aboard! Default currency is USD, to change send `/currency RUB`",
				),
			)
			return nil
		}
		return b.handleError(msg.Chat.ID, &UserNotFoundError{})
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
		_, updated, err := b.storage.SaveEntry(ctx, user, &cmd.(*EntryCommand).Entry)
		if err != nil {
			return b.handleError(msg.Chat.ID, err)
		}
		if !updated {
			_, _ = b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "Added")) // TODO: i18n, return sum and currency
		}
		// TODO: if updated edit message
		// TODO: save reply id to edit if original message was edited
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

func New(token string, logger *logrus.Logger, storage Storage) (*Bot, error) {
	return &Bot{
		logger:  logger,
		token:   token,
		storage: storage,
		stopC:   make(chan struct{}, 1),
		doneC:   make(chan struct{}, 1),
	}, nil
}