package app

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/MarlikAlmighty/analyze-it/internal/config"
	"github.com/MarlikAlmighty/analyze-it/internal/models"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/go-redis/redis/v8"
	tg "gopkg.in/telegram-bot-api.v4"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Core application
type Core struct {
	Config Config `config:"-"`
	Store  Store  `store:"-"`
	Server Server `server:"-"`
}

// New application app initialization
func New(c *config.Configuration, r Store, s *http.Server) *Core {
	return &Core{
		Config: c,
		Store:  r,
		Server: s,
	}
}

type (
	App interface {
		Run()
		Stop()
		getLinkRzn() (map[string]string, error)
		catchPostFromRzn(m map[string]string) (*models.Array, error)
		getLinkYa() (map[string]string, error)
		catchPostFromYa(m map[string]string) (models.Array, error)
		checkLink(m map[string]string) (map[string]string, error)
		checkPreSend(arr models.Array) error
		findWords(title, body string) string
		senToTelegram(p models.Post) error
		browser(url string) (string, error)
		createMD5Hash(s string) string
	}
	Config interface {
	}
	Server interface {
		ListenAndServe() error
		Shutdown(ctx context.Context) error
	}
	Store interface {
		Get(ctx context.Context, key string) *redis.StringCmd
		Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
		Close() error
	}
)

func (core *Core) Run() {

	var err error

	for {

		m := make(map[string]string)

		if m, err = core.getLinkRzn(); err != nil {
			log.Println("[ERROR] get link from rzn: " + err.Error())
			break
		}

		if len(m) > 0 {

			log.Printf("got links from rzn: %v\n", len(m))

			mp := make(map[string]string)
			if mp, err = core.checkLink(m); err != nil {
				log.Printf("catch link from rzn: %v \n", err)
				break
			}

			var data models.Array
			if data, err = core.catchPostFromRzn(mp); err != nil {
				log.Println("[ERROR] catch post from rzn: " + err.Error())
				break
			}

			if err = core.checkPreSend(data); err != nil {
				log.Println("[ERROR] check key words: " + err.Error())
				break
			}

			log.Println("Finished send post from rzn")
		}

		if m, err = core.getLinkYa(); err != nil {
			log.Printf("[ERROR] get link from ya: " + err.Error())
			break
		}

		if len(m) > 0 {

			log.Printf("got links from ya: %v\n", len(m))

			mp := make(map[string]string)
			if mp, err = core.checkLink(m); err != nil {
				log.Printf("catch link from rzn: %v \n", err)
				break
			}

			var data models.Array
			if data, err = core.catchPostFromYa(mp); err != nil {
				log.Println("[ERROR] catch post from ya: " + err.Error())
				break
			}

			if err = core.checkPreSend(data); err != nil {
				log.Println("[ERROR] check key words: " + err.Error())
				break
			}

			log.Println("Finished send post from ya")
		}

		time.Sleep(1 * time.Hour)
	}
}

func (core *Core) browser(url string) (string, error) {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("window-size", "1,1"),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancelTask()

	ctx, cancel := context.WithTimeout(taskCtx, 1*time.Minute)
	defer cancel()

	var html string

	log.Println("got url " + url)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return html, nil
}

func (core *Core) getLinkRzn() (map[string]string, error) {

	if core.Config.(*config.Configuration).RznUrl == "" {
		return nil, errors.New("error get RZN URL from env")
	}

	var (
		html string
		err  error
	)
	m := make(map[string]string)

	log.Println("send url " + core.Config.(*config.Configuration).RznUrl)

	if html, err = core.browser(core.Config.(*config.Configuration).RznUrl); err != nil {
		return nil, err
	}

	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
		return nil, err
	}

	doc.Find("#news-container > .stories .stories-item__title > a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		title := s.Text()
		m[link] = title
	})

	return m, nil
}

func (core *Core) catchPostFromRzn(m map[string]string) (models.Array, error) {

	log.Println("start catch posts from rzn")

	reg := regexp.MustCompile(`\s+`)

	data := models.Array{}

	var (
		html string
		err  error
	)

	for k := range m {

		post := models.Post{}

		if html, err = core.browser(k); err != nil {
			log.Println("[ERROR] catch post from rzn")
			break
		}

		var doc *goquery.Document
		if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
			log.Println("[ERROR] catch post from rzn")
			break
		}

		var (
			title, link, img string
			ex               bool
		)

		doc.Find("#newsContainer > div.row.url-checkpoint.newsItem.story > div.col.story__details > div > div.story__body > div.story__hero > div > img").Each(func(i int, s *goquery.Selection) {
			img, ex = s.Attr("src")
			if !ex {
				log.Println("img not found")
			}
		})

		doc.Find("#newsContainer > div.row.url-checkpoint.newsItem.story").Each(func(i int, s *goquery.Selection) {
			title, ex = s.Attr("data-title")
			if !ex {
				log.Println("title not found")
			}
			link, ex = s.Attr("data-url")
			if !ex {
				log.Println("link not found")
			}
		})

		var (
			str  []string
			body string
			txt  string
		)

		doc.Find("#newsContainer > div.row.url-checkpoint.newsItem.story > div.col.story__details > div > div.story__body > div:nth-child(3)").Each(func(i int, s *goquery.Selection) {
			txt = s.Text()
			newTxt := reg.ReplaceAllString(txt, " ")
			str = append(str, newTxt)
		})

		body = strings.Join(str, "")
		post.Hash = core.getMD5Hash(title)
		post.Title = title
		post.Body = body
		post.Image = img
		post.Link = link
		data = append(data, &post)

		log.Printf("append post %v\n", post.Link)

		time.Sleep(10 * time.Second)
	}

	return data, nil
}

func (core *Core) getLinkYa() (map[string]string, error) {

	if core.Config.(*config.Configuration).YaUrl == "" {
		return nil, errors.New("error get YA URL from env")
	}

	var (
		html string
		err  error
	)

	m := make(map[string]string)

	log.Println("send url " + core.Config.(*config.Configuration).YaUrl)
	if html, err = core.browser(core.Config.(*config.Configuration).YaUrl); err != nil {
		return nil, err
	}

	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
		return nil, err
	}
	doc.Find("div.item a.subject").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		hyperlink := "https://ya62.ru" + link
		title := s.Text()
		m[hyperlink] = title
	})

	return m, nil
}

func (core *Core) catchPostFromYa(m map[string]string) (models.Array, error) {

	space := regexp.MustCompile(`[[:space:]]`)
	all := regexp.MustCompile(`\s+`)

	data := models.Array{}

	var (
		html string
		err  error
	)

	for link, title := range m {

		post := models.Post{}

		if html, err = core.browser(link); err != nil {
			log.Println("[ERROR] catch post from rzn")
			break
		}

		var doc *goquery.Document
		if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
			log.Println("[ERROR] catch post from rzn")
			break
		}

		var txt, img string

		doc.Find("figure > img").Each(func(i int, s *goquery.Selection) {
			img, _ = s.Attr("data-lrg")
			img = "https://ya62.ru" + img
		})

		doc.Find("div.news-detail p").Each(func(i int, s *goquery.Selection) {
			txt += s.Text()
			txt = space.ReplaceAllString(txt, " ")
			txt = all.ReplaceAllString(txt, " ")
			txt = strings.TrimSpace(txt)
		})

		post.Hash = core.getMD5Hash(title)
		post.Title = title
		post.Body = txt
		post.Image = img
		post.Link = link
		data = append(data, &post)

		log.Printf("append post %v\n", post.Link)

		time.Sleep(10 * time.Second)
	}

	return data, nil
}

func (core *Core) checkLink(m map[string]string) (map[string]string, error) {

	log.Println("start check link")

	mp := make(map[string]string)

	for k, v := range m {
		hash := core.getMD5Hash(k)
		if _, err := core.Store.Get(context.Background(), hash).Result(); err == redis.Nil {
			mp[k] = v
		} else if err != nil {
			return nil, err
		}
	}

	return mp, nil
}

func (core *Core) checkPreSend(arr models.Array) error {

	log.Println("start check key word function")

	for _, v := range arr {

		if v.Title == "" || v.Body == "" || v.Image == "" || v.Hash == "" {
			log.Println("[ERROR] any fields of struct post is not defined")
			break
		}

		keyWord := core.findWords(v.Title, v.Body)

		if len(keyWord) > 0 {

			if err := core.senToTelegram(*v); err != nil {
				return err
			}

			if err := core.Store.Set(context.Background(), v.Hash, v.Title, 48*time.Hour).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (core *Core) senToTelegram(p models.Post) error {

	var (
		botAPI  *tg.BotAPI
		channel int64
		err     error
	)

	if channel, err = strconv.ParseInt(core.Config.(*config.Configuration).Channel, 10, 64); err != nil {
		return err
	}

	if botAPI, err = tg.NewBotAPI(core.Config.(*config.Configuration).BotToken); err != nil {
		log.Fatalln(err)
	}

	text := fmt.Sprintf("<b>%s</b> \n\n %s \n <a href='%s'>&#8203;</a>", p.Title, p.Body, p.Image)

	msg := tg.NewMessage(channel, text)

	msg.ParseMode = "HTML"

	if _, err = botAPI.Send(msg); err != nil {
		return err
	}

	return nil
}

// Stop app, shutdown server, close connection
func (core *Core) Stop() {

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := core.Store.Close(); err != nil {
		log.Println(err)
	}

	if err := core.Server.Shutdown(ctx); err != nil {
		log.Println(err)
	}
}

// findWords find in text a key words
func (core *Core) findWords(title, body string) string {

	var whatWords = "гибдд угибдд дпс пдд мчс мвд фсб умвд м5 м6 " +
		"инспектор автоинспектор полицейски полици пристав приговор " +
		"суд осуд осуж уголовн оштрафо арест дтп авари столкновени " +
		"столкнул врезал протаранил протаранивш притё опрокинул " +
		"опрокидыв перевернул провали перелет улёт улет влете обрушивш " +
		"рухнул снёс снес въехал падени упа сбил сби расстерзал разорвал " +
		"оборва загрыз прорвал бомб снаряд оружи боеприпа мин укус рейд " +
		"взрыв взорвал убий гибел убил зареза скончал борьб драк побоищ " +
		"расстреля подрал нарко пожар загорел сгорел возгоран горит горел " +
		"тело труп мёртвы мёртво мертво мертве умер поги гибел гибн " +
		"смертельн пропал упал выпа эвакуа утону утоп смыл затопил сгоре " +
		"ограб грабит разбой разбойни мошенн обманул фальшив кража укра " +
		"пропажа пропа угон обго угнал ищут поиск розыск разыскива отрави " +
		"вскры взлома насил изнасил бешенств взятк суд коррупци беспредел " +
		"напал нападени разборк преследова сбежав сбежал протест забастовк " +
		"бастовал бунт такс пострада подозрит проституц проститут погода " +
		"дождь гроза ветер туман опасност заморозки похолода циклон урага " +
		"снег синопти холод футбол хоккей хоккеи игрок нетрезв пьян рязан "

	Title := strings.Fields(title)
	Body := strings.Fields(body)
	What := strings.Fields(whatWords)
	reg := regexp.MustCompile(`[а-яА-Я]{1,6}`)

	for _, what := range What {
		for _, title = range Title {
			for _, body = range Body {
				if strings.EqualFold(reg.FindString(strings.ToLower(what)), reg.FindString(strings.ToLower(title))) ||
					strings.EqualFold(reg.FindString(strings.ToLower(what)), reg.FindString(strings.ToLower(body))) {
					return what
				}
			}
		}
	}
	return ""
}

func (core *Core) getMD5Hash(s string) string {
	md := md5.New()
	md.Write([]byte(s))
	return hex.EncodeToString(md.Sum(nil))
}
