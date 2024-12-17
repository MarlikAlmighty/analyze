package app

import (
	"errors"
	"regexp"
	"strings"

	"github.com/MarlikAlmighty/analyze-it/internal/models"
	"github.com/PuerkitoBio/goquery"
)

func (core *Core) getLinkYa(html string) (map[string]string, error) {

	var (
		doc *goquery.Document
		err error
	)

	if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
		return nil, err
	}

	m := make(map[string]string)
	doc.Find("div.bqFI3 > div > a.OTasl").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		hyperlink := "https://ya62.ru" + link
		title := s.Text()
		m[hyperlink] = title
	})

	if len(m) == 0 {
		return nil, errors.New("links is zero")
	}

	return m, nil
}

func (core *Core) catchPostFromYa(html, link string) (models.Post, error) {

	space := regexp.MustCompile(`[[:space:]]`)
	all := regexp.MustCompile(`\s+`)
	tag := regexp.MustCompile(`[<\.+>]`)
	post := models.Post{}

	var (
		doc  *goquery.Document
		body string
		err  error
	)

	if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
		return post, errors.New(err.Error())
	}

	doc.Find("div.news-detail > h1").Each(func(i int, s *goquery.Selection) {
		post.Title = s.Text()
	})

	doc.Find("div.news-detail > figure > img").Each(func(i int, s *goquery.Selection) {
		tmp, _ := s.Attr("src")
		post.Image = "https://ya62.ru" + tmp
	})

	doc.Find("div.news-detail p").Each(func(i int, s *goquery.Selection) {
		body += s.Text()
	})

	body = space.ReplaceAllString(body, " ")
	body = all.ReplaceAllString(body, " ")
	body = strings.Replace(body, "<...>", "", -1)
	body = tag.ReplaceAllString(body, "")
	body = strings.Replace(body, "YA62ru", "", 3)
	body = strings.TrimSpace(body)

	post.Hash = core.stringToHash(link)
	post.Body = body
	post.Link = link

	return post, nil
}
