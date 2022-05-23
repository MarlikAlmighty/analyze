package app

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MarlikAlmighty/analyze-it/internal/config"
	"github.com/MarlikAlmighty/analyze-it/internal/models"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	tg "gopkg.in/telegram-bot-api.v4"
)

// Core application
type Core struct {
	Config Config `config:"-"`
	Store  Store  `store:"-"`
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
		getLinkRzn() (map[string]string, error)
		catchPostFromRzn(m map[string]string) (*models.Array, error)
		getLinkYa() (map[string]string, error)
		catchPostFromYa(m map[string]string) (models.Array, error)
		checkLink(m map[string]string) (map[string]string, error)
		checkPreSend(arr models.Array) error
		findWords(title, body string) string
		sendToTelegram(p models.Post) error
		browser(url string) (string, error)
		createMD5Hash(s string) string
		mustParseDuration(s string) time.Duration
	}
	Config interface {
	}
	Store interface {
		Get(ctx context.Context, key string) *redis.StringCmd
		Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
		Close() error
	}
)

func (core *Core) Run() {

	statsInt := core.mustParseDuration("1h")
	statsTimer := time.NewTimer(statsInt)
	rm := make(map[string]string)
	ym := make(map[string]string)
	var err error

	for {

		statsTimer.Reset(statsInt)

		go func() {

			if rm, err = core.getLinkRzn(); err != nil {
				log.Println("[ERROR] get link from rzn: " + err.Error())
				return
			}

			log.Printf("got links from rzn: %v\n", len(rm))
			if len(rm) == 0 {
				return
			}

			data := models.Array{}
			if data, err = core.catchPostFromRzn(rm); err != nil {
				log.Println("[ERROR] catch post from rzn: " + err.Error())
				return
			}

			if err = core.checkPreSend(data); err != nil {
				log.Println("[ERROR] check key words: " + err.Error())
				return
			}

			log.Println("Finished send post from rzn")
		}()

		go func() {

			if ym, err = core.getLinkYa(); err != nil {
				log.Printf("[ERROR] get link from ya: " + err.Error())
				return
			}

			log.Printf("got links from ya: %v\n", len(ym))
			if len(ym) == 0 {
				return
			}

			data := models.Array{}
			if data, err = core.catchPostFromYa(ym); err != nil {
				log.Println("[ERROR] catch post from ya: " + err.Error())
				return
			}

			if err = core.checkPreSend(data); err != nil {
				log.Println("[ERROR] check key words: " + err.Error())
				return
			}

			log.Println("Finished send post from ya")
		}()

		<-statsTimer.C
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

	ctx, cancel := context.WithTimeout(taskCtx, 2*time.Minute)
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
			continue
		}

		var doc *goquery.Document
		if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
			return nil, errors.New(err.Error())
		}

		var (
			title, link, img string
		)

		doc.Find("#newsContainer > div.row.url-checkpoint.newsItem.story > div.col.story__details > div > div.story__body > div.story__hero > div > img").Each(func(i int, s *goquery.Selection) {
			img, _ = s.Attr("src")
		})

		doc.Find("#newsContainer > div.row.url-checkpoint.newsItem.story").Each(func(i int, s *goquery.Selection) {
			title, _ = s.Attr("data-title")
			link, _ = s.Attr("data-url")
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
			log.Printf("%v\n", err)
			continue
		}

		var doc *goquery.Document
		if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
			return nil, errors.New(err.Error())
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
			txt = all.ReplaceAllString("YA62.ru", "")
			txt = strings.TrimSpace(txt)

		})

		post.Hash = core.getMD5Hash(title)
		post.Title = title
		post.Body = txt
		post.Image = img
		post.Link = link
		data = append(data, &post)

		time.Sleep(10 * time.Second)
	}

	return data, nil
}

func (core *Core) checkPreSend(arr models.Array) error {

	log.Printf("start presend func, length obj: %v\n", len(arr))

	var (
		keyWord string
		count   int
	)

	for _, v := range arr {

		if v.Title == "" || v.Body == "" || v.Hash == "" || v.Link == "" || v.Image == "" {
			log.Printf("[ERROR] any fields is not defined %v\n", v.Link)
			continue
		}

		if _, err := core.Store.Get(context.Background(), v.Hash).Result(); err == redis.Nil {

			keyWord = core.findWords(v.Body)

			if len(keyWord) > 0 {

				if err = core.sendToTelegram(*v); err != nil {
					return err
				}

				count++

				if err = core.Store.Set(context.Background(), v.Hash, v.Title, 48*time.Hour).Err(); err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		}
	}

	log.Printf("send %v post to telegram\n", count)
	return nil
}

func (core *Core) sendToTelegram(p models.Post) error {

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

func (core *Core) Stop() {

	if err := core.Store.Close(); err != nil {
		log.Println(err)
	}
}

// findWords find in text a keywords
func (core *Core) findWords(body string) string {

	var whatWords = "гибдд угибдд дпс пдд мчс мвд фсб умвд лиза алерт" +
		"инспектор автоинспектор полицейски полици пристав" +
		"суд осуд осуж уголовн оштрафо штраф арест взятк коррупци беспредел" +
		"пропажа пропа приговор ищут поиск розыск разыскива" +
		"дтп авари столкновени столкнул врезал протаранил протаранивш притё угон обго угнал" +
		"опрокинул вылете опрокидыв перевернул провали перелет улёт улет влете обогн" +
		"обрушивш рухнул снёс снес въехал падени упа сбил травм м5 м6" +
		"бомб снаряд оружи боеприпа минирова укус рейд взрыв взорвал" +
		"смертельн пропал упал выпа эвакуа утону утоп смыл затопил сгоре" +
		"пожар загорел сгорел возгоран гори горело горела оборва прорвал" +
		"убий гибел убил зареза скончал борьб драк побоищ отрави" +
		"изби расстерзал разорвал расстреля подрал нарко загрыз" +
		"тело труп мёртвы мёртво мертво мертве умер поги гибел гибн" +
		"напал нападени разборк преследова сбежав сбежал" +
		"ограб грабит разбой разбойни мошенн обманул фальшив краж укра" +
		"вскры взлома насил изнасил бешенств нетрезв пьян рязан" +
		"протест забастов бастовал пострада подозрит проституц проститут" +
		"дождь гроза ветер туман уровень желтый красный опасност" +
		"заморозки похолода циклон урага снег синопти холод погод"

	Body := strings.Fields(body)
	What := strings.Fields(whatWords)
	reg := regexp.MustCompile(`[а-яА-Я]{1,6}`)

	for _, what := range What {
		for _, body = range Body {
			if strings.EqualFold(reg.FindString(strings.ToLower(what)), reg.FindString(strings.ToLower(body))) {
				return "yes" // what
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

func (core *Core) mustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		log.Fatal(err)
	}
	return value
}
