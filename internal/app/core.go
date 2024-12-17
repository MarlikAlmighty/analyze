package app

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/MarlikAlmighty/analyze-it/internal/config"
	"github.com/MarlikAlmighty/analyze-it/internal/models"
	"github.com/sclevine/agouti"
)

// Core application
type Core struct {
	Config *config.Configuration `config:"-"`
	Store  Store                 `store:"-"`
}

// New application app initialization
func New(c *config.Configuration, r Store) *Core {
	return &Core{
		Config: c,
		Store:  r,
	}
}

type (
	App interface {
		Run()
		Stop()
		//browser(opts []chromedp.ExecAllocatorOption, url string) (string, error)
		checkLink(m map[string]string) (map[string]string, error)
		checkPreSend(v models.Post) error
		checkBlank(v *models.Post) bool
		sendToModerChannel(p *models.Post) error
		stringToHash(s string) string
		mustParseDuration(s string) time.Duration
		getLinkRzn(html string) (map[string]string, error)
		catchPostFromRzn(html string) (models.Post, error)
		getLinkYa(html string) (map[string]string, error)
		catchPostFromYa(html, link string) (models.Post, error)
	}
	Store interface {
		Write(bucket, key string, value []byte) error
		Read(bucket, key string) ([]byte, error)
		Sweep(maxAge time.Duration) error
		GetExpired(maxAge time.Duration) ([][]byte, error)
		Close() error
	}
)

func (core *Core) Run() {

	statsInt := core.mustParseDuration("30m")
	statsTimer := time.NewTimer(statsInt)
	mp := make(map[string]string)

	var (
		page *agouti.Page
		post models.Post
		html string
		err  error
	)

	// init driver
	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{
			"--headless",
			"--disable-gpu",
			"--no-sandbox",
			"--whitelisted-ips",
		}),
		// agouti.Debug,
	)

	// ttl all posts
	maxAge := 168 * time.Hour

	for {

		statsTimer.Reset(statsInt)

		if err = driver.Start(); err != nil {
			log.Printf("error driver start: %v\n", err)
			return
		}

		func() {
			log.Println("start clear database")
			if err = core.Store.Sweep(maxAge); err != nil {
				log.Printf("error sweep: %v\n", err)
				return
			}
		}()

		page, err = driver.NewPage()
		if err != nil {
			log.Println("[YA62]: error new page")
		}

		defer func() {
			if err = page.Destroy(); err != nil {
				log.Println("[RZN]: error page destroy")
			}
		}()

		/*

			log.Printf("start parsing %s", core.Config.RznUrl)

			if err = page.Navigate(core.Config.RznUrl); err != nil {
				log.Println("[RZN]: error got main page: " + err.Error())
				return
			}

			if html, err = page.HTML(); err != nil {
				log.Println("[RZN]: error got html: " + err.Error())
				return
			}

			// got all links
			if mp, err = core.getLinkRzn(html); err != nil {
				log.Println("[RZN]: error got links from rzn: " + err.Error())
				return
			}

			if mp, err = core.checkLink(mp); err != nil {
				log.Println("[RZN]: error check link from rzn: " + err.Error())
				return
			}

			// range for links
			for url := range mp {

				time.Sleep(10 * time.Second)

				if err = page.Navigate(url); err != nil {
					log.Println("[RZN]: error got links page: " + err.Error())
					continue
				}

				if html, err = page.HTML(); err != nil {
					log.Println("[RZN]: error got html: " + err.Error())
					return
				}

				// catch title, post, image from link
				if post, err = core.catchPostFromRzn(html); err != nil {
					log.Println("[RZN]: error catch post from rzn: " + err.Error())
					continue
				}

				// check and send post
				if err = core.checkPreSend(post); err != nil {
					log.Println("[RZN]: sender error: " + err.Error())
					continue
				}

			}
		*/

		log.Printf("start parsing %s", core.Config.YaUrl)

		// get start page ya62.ru/text/incidents/
		if err = page.Navigate(core.Config.YaUrl); err != nil {
			log.Println("[YA62]: error got main page: " + err.Error())
			return
		}

		if html, err = page.HTML(); err != nil {
			log.Println("[YA62]: error got html: " + err.Error())
			return
		}

		// got all links
		if mp, err = core.getLinkYa(html); err != nil {
			log.Printf("[YA62]: error got link from ya: " + err.Error())
			return
		}

		if mp, err = core.checkLink(mp); err != nil {
			log.Printf("[YA62]: error check link from ya: " + err.Error())
			return
		}

		// range for links
		for url := range mp {

			time.Sleep(10 * time.Second)

			if err = page.Navigate(url); err != nil {
				log.Println("[YA62]: error got target page: " + err.Error())
				return
			}

			if html, err = page.HTML(); err != nil {
				log.Println("[YA62]: error got html: " + err.Error())
				return
			}

			// catch title, post, image from link
			if post, err = core.catchPostFromYa(html, url); err != nil {
				log.Println("[YA62]: error catch post from ya: " + err.Error())
				continue
			}

			// check and send post
			if err = core.checkPreSend(post); err != nil {
				log.Println("[YA62]: sender error: " + err.Error())
				continue
			}
		}

		if err = page.Destroy(); err != nil {
			log.Println("[YA62]: error page destroy")
		}

		if err = driver.Stop(); err != nil {
			log.Printf("error driver stop: %v\n", err)
		}

		log.Println("Timeout 30 minutes...")

		<-statsTimer.C
	}
}

func (core *Core) checkLink(m map[string]string) (map[string]string, error) {

	var (
		b    []byte
		hash string
		err  error
	)

	newMap := make(map[string]string)

	for k, v := range m {

		hash = core.stringToHash(k)

		if b, err = core.Store.Read("posts", hash); err != nil {
			log.Printf("error read from db: %v\n", err)
			panic(err)
		}

		if len(b) == 0 {
			newMap[k] = v
		}
	}

	return newMap, nil
}

func (core *Core) checkPreSend(v models.Post) error {

	var err error

	// checking for missing fields in a structure
	if core.checkBlank(&v) {
		return nil
	}

	if err = core.sendToModerChannel(&v); err != nil {
		return err
	}

	// marshal post
	var post []byte
	if post, err = json.Marshal(v); err != nil {
		return err
	}

	// write post to database
	if err = core.Store.Write("posts", v.Hash, post); err != nil {
		return err
	}

	// writing ttl posts to database
	if err = core.Store.Write("ttl", time.Now().UTC().Format(time.RFC3339Nano),
		[]byte(v.Hash)); err != nil {
		return err
	}

	return nil
}

func (core *Core) sendToModerChannel(p *models.Post) error {
	var (
		botAPI *tgbotapi.BotAPI
		err    error
	)
	if botAPI, err = tgbotapi.NewBotAPI(core.Config.BotToken); err != nil {
		return err
	}
	text := fmt.Sprintf("<b>%s</b> \n\n %s \n <a href='%s'>&#8203;</a>", p.Title, p.Body, p.Image)
	msg := tgbotapi.NewMessage(core.Config.ModeratorChannel, text)
	msg.ParseMode = "HTML"
	mp := [][]string{
		{"Post", p.Hash},
		{"Remove", "Remove"},
	}
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(mp))
	for _, value := range mp {
		row := tgbotapi.NewInlineKeyboardRow()
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(value[0], value[1]))
		rows = append(rows, row)
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg.ReplyMarkup = &keyboard
	if _, err = botAPI.Send(msg); err != nil {
		return err
	}
	return nil
}

func (core *Core) stringToHash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (core *Core) mustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		log.Fatal(err)
	}
	return value
}

func (core *Core) checkBlank(v *models.Post) bool {
	if v.Title == "" || v.Body == "" || v.Hash == "" || v.Link == "" || v.Image == "" {
		return true
	}
	return false
}

func (core *Core) Stop() {
	if err := core.Store.Close(); err != nil {
		log.Println(err)
	}
}
