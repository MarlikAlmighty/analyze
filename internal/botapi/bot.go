package botapi

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/MarlikAlmighty/analyze-it/internal/config"
	"github.com/MarlikAlmighty/analyze-it/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TgAPI struct {
	Config *config.Configuration `config:"-"`
	Store  Store                 `store:"-"`
}

type (
	Store interface {
		Read(bucket, key string) ([]byte, error)
	}
)

// New application app initialization
func New(cnf *config.Configuration, r Store) (*TgAPI, error) {
	return &TgAPI{
		Config: cnf,
		Store:  r,
	}, nil
}

func (app *TgAPI) Run() error {

	// Start botAPI with token
	bot, err := tgbotapi.NewBotAPI(app.Config.BotToken)
	if err != nil {
		return err
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {

		if update.CallbackQuery != nil {

			switch update.CallbackQuery.Data {

			case "Remove":

				// send callback
				if _, err = bot.Request(tgbotapi.CallbackConfig{
					CallbackQueryID: update.CallbackQuery.ID,
				}); err != nil {
					log.Printf("error answer callback: %v\n", err)
				}

				// deleting message from moder channel
				if _, err = bot.Request(tgbotapi.DeleteMessageConfig{
					ChatID:    update.CallbackQuery.Message.Chat.ID,
					MessageID: update.CallbackQuery.Message.MessageID,
				}); err != nil {
					log.Printf("error delete message: %v\n", err)
				}

			default:

				var (
					p   *models.Post
					b   []byte
					err error
				)

				// send callback
				if _, err = bot.Request(tgbotapi.CallbackConfig{
					CallbackQueryID: update.CallbackQuery.ID,
				}); err != nil {
					log.Printf("error answer callback: %v\n", err)
				}

				// read from database post
				if b, err = app.Store.Read("posts", update.CallbackQuery.Data); err != nil {
					log.Printf("error read from database: %v\n", err)
				}

				// if we have post, unmarshal his
				if len(b) > 0 {
					if err = json.Unmarshal(b, &p); err != nil {
						log.Printf("error unmarshal model: %v\n", err)
					}
				}

				// send post to main channel
				text := fmt.Sprintf("<b>%s</b> \n\n %s \n <a href='%s'>&#8203;</a>", p.Title, p.Body, p.Image)
				msg := tgbotapi.NewMessage(app.Config.MainChannel, text)
				msg.ParseMode = "HTML"
				if _, err = bot.Send(msg); err != nil {
					log.Printf("error send post: %v\n", err)
				}

				// then, deleting message from moder channel
				deleteMessageConfig := tgbotapi.DeleteMessageConfig{
					ChatID:    update.CallbackQuery.Message.Chat.ID,
					MessageID: update.CallbackQuery.Message.MessageID,
				}

				if _, err = bot.Request(deleteMessageConfig); err != nil {
					log.Printf("error delete message: %v\n", err)
				}
			}
		}
	}

	return nil
}
