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
	doc.Find(".textBox_fgrum > a.header_fgrum").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		hyperlink := link
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

	doc.Find("h1.title_ip27z").Each(func(i int, s *goquery.Selection) {
		post.Title = s.Text()
	})

	doc.Find("div.imageWrapper_nZVrb > picture > img").Each(func(i int, s *goquery.Selection) {
		post.Image, _ = s.Attr("src")
	})

	doc.Find("div.uiArticleBlockText_g83x5").Each(func(i int, s *goquery.Selection) {
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
