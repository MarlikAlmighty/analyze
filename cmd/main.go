package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MarlikAlmighty/analyze-it/models"
	"github.com/MarlikAlmighty/analyze-it/sort"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-redis/redis/v8"
	tg "gopkg.in/telegram-bot-api.v4"
)

func main() {

	var ctx = context.Background()

	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalln(err)
	}

	r := redis.NewClient(&redis.Options{
		Addr:     opt.Addr,
		Password: opt.Password,
		DB:       opt.DB,
	})

	if _, err := r.Ping(ctx).Result(); err != nil {
		log.Fatalln(err)
	}

	go parse(r, ctx)

	router := mux.NewRouter()

	router.HandleFunc("/", homeHandler)

	log.Printf("Start serving on :%s \n", os.Getenv("PORT"))

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))

}

func parse(r *redis.Client, ctx context.Context) {

	for {

		obj := models.Obj{}
		data := models.Array{}

		statsInt := MustParseDuration("30m")
		statsTimer := time.NewTimer(statsInt)

		select {

		case <-statsTimer.C:

			data = models.Array{}

			mp, err := getLinkYa()
			if err != nil {
				log.Println("Error get link from ya")
			}

			if len(mp) > 0 {
				data, err = addPostFromYa(mp, data)
				if err != nil {
					log.Println("Error get content from ya")
				}
			}

			m, err := getLinkRzn()
			if err != nil {
				log.Println("Error get link from rzn")
			}

			if len(m) > 0 {
				data, err = addPostFromRzn(m, data)
				if err != nil {
					log.Println("Error get content from rzn")
				}
			}

			t := time.Now().UTC()

			obj = models.Obj{}

			obj.Timestamp = t.String()

			obj.Array = data

			sortObj(ctx, r, obj.Array)

			statsTimer.Reset(statsInt)
		}
	}
}

func homeHandler(w http.ResponseWriter, _ *http.Request) {
	if _, err := fmt.Fprintf(w, "I'm alive!"); err != nil {
		log.Fatalln(err)
	}
}

func sortObj(ctx context.Context, r *redis.Client, arr models.Array) {

	for _, v := range arr {

		if _, err := r.Get(ctx, v.Hash).Result(); err == redis.Nil {

			if len(v.Image) <= 0 {
				break
			}

			keyWord := sort.FindWords(v.Title, v.Body)

			if len(keyWord) > 0 {
				postChanel(v)

				if err := r.Set(ctx, v.Hash, v.Title, 48*time.Hour).Err(); err != nil {
					log.Fatalln(err)
				}
			}

		} else if err != nil {

			log.Fatalln(err)

		}

	}
}

func postChanel(p *models.Post) {

	botAPI := os.Getenv("BOT_TOKEN")
	channel := os.Getenv("CHANNEL")

	digitChannel, err := strconv.ParseInt(channel, 10, 64)
	if err != nil {
		log.Fatalln(err)
	}

	bot, err := tg.NewBotAPI(botAPI)
	if err != nil {
		log.Fatalln(err)
	}

	text := fmt.Sprintf("<b>%s</b> \n\n %s \n <a href='%s'>&#8203;</a>", p.Title, p.Body, p.Image)

	msg := tg.NewMessage(digitChannel, text)

	msg.ParseMode = "HTML"

	if _, err := bot.Send(msg); err != nil {
		log.Fatalln(err)
	}
}

func getMD5Hash(s string) string {

	md := md5.New()

	md.Write([]byte(s))

	return hex.EncodeToString(md.Sum(nil))
}

func MustParseDuration(s string) time.Duration {

	value, err := time.ParseDuration(s)

	if err != nil {
		log.Fatalln(err)
	}

	return value
}

func addPostFromRzn(m map[string]string, arr models.Array) (models.Array, error) {

	reg := regexp.MustCompile(`\s+`)

	for link, title := range m {

		var res *http.Response

		res, err := http.Get(link)
		if err != nil {
			return nil, err
		}

		if res.StatusCode != 200 {
			return nil, err
		}

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			return nil, err
		}

		var imgLink string

		doc.Find("div.item-news-canvas__wrapper-img img").Each(func(i int, s *goquery.Selection) {
			img, _ := s.Attr("src")
			imgLink = img
		})

		var (
			str  []string
			body string
		)

		doc.Find("div.text p").Each(func(i int, s *goquery.Selection) {
			txt := s.Text()
			newTxt := reg.ReplaceAllString(txt, " ")
			str = append(str, newTxt)
		})

		body = strings.Join(str, "")

		post := models.Post{}

		post.Hash = getMD5Hash(title)

		post.Title = title

		post.Body = body

		post.Image = imgLink

		post.Link = link

		err = res.Body.Close()
		if err != nil {
			log.Fatalln(err)
		}

		num := strings.Index(post.Body, `©`)

		post.Body = body[0:num]

		arr = append(arr, &post)
	}
	return arr, nil
}

func getLinkRzn() (map[string]string, error) {

	rzn := os.Getenv("RZN_URL")
	if rzn == "" {
		log.Fatalln("Error get rzn url from env")
	}

	var m = make(map[string]string)

	res, err := http.Get(rzn)
	if err != nil {
		return m, err
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("close body err: %v\n", err)
		}
	}()

	if res.StatusCode != 200 {
		return m, err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	doc.Find("div.headerLinks.headerLinks_style_custom.text.thumb-middle a.bLink").Each(func(i int, s *goquery.Selection) {

		link, _ := s.Attr("href")

		title, _ := s.Attr("title")

		m[link] = title
	})

	return m, err
}

func getLinkYa() (map[string]string, error) {

	ya := os.Getenv("YA_URL")
	if ya == "" {
		log.Fatalln("Error get ya url from env")
	}

	var m = make(map[string]string)

	res, err := http.Get(ya)
	if err != nil {
		return m, err
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("close body err: %v\n", err)
		}
	}()

	if res.StatusCode != 200 {
		return m, err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return m, err
	}

	doc.Find("div.item a.subject").Each(func(i int, s *goquery.Selection) {

		link, _ := s.Attr("href")

		hyperlink := "https://ya62.ru" + link

		title := s.Text()

		m[hyperlink] = title
	})

	return m, err
}

func addPostFromYa(m map[string]string, arr models.Array) (models.Array, error) {

	space := regexp.MustCompile(`[[:space:]]`)

	all := regexp.MustCompile(`\s+`)

	for link, title := range m {

		var res *http.Response

		res, err := http.Get(link)
		if err != nil {
			return nil, err
		}

		if res.StatusCode != 200 {
			return nil, err
		}

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			return nil, err
		}

		var imgLink string

		doc.Find("div.news-detail .news_detail_img img").Each(func(i int, s *goquery.Selection) {
			img, _ := s.Attr("data-lrg")
			imgLink = "http://opt-727797.ssl.1c-bitrix-cdn.ru" + img
		})

		var (
			str  []string
			body string
		)

		doc.Find("div.news-detail p").Each(func(i int, s *goquery.Selection) {

			txt := s.Text()

			newTxt := space.ReplaceAllString(txt, " ")

			newTxt = all.ReplaceAllString(newTxt, " ")

			newTxt = strings.TrimSpace(newTxt)

			str = append(str, newTxt)
		})

		body = strings.Join(str, "")

		post := models.Post{}

		post.Hash = getMD5Hash(title)

		post.Title = title

		post.Body = body

		post.Image = imgLink

		post.Link = link

		if err := res.Body.Close(); err != nil {
			log.Printf("close body err: %v\n", err)
		}

		arr = append(arr, &post)
	}

	return arr, nil
}
