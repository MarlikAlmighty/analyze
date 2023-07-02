package app

import (
	"errors"
	"github.com/MarlikAlmighty/analyze-it/internal/models"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
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
	doc.Find("div.home-top__slide > div.news-card > div.news-card__info > a").Each(func(i int, s *goquery.Selection) {
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
	post := models.Post{}

	var (
		doc *goquery.Document
		err error
	)

	if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
		return post, errors.New(err.Error())
	}

	var title, txt, img string

	doc.Find("div.news-detail > h1").Each(func(i int, s *goquery.Selection) {
		title = s.Text()
	})

	doc.Find("div.news-detail > figure > img").Each(func(i int, s *goquery.Selection) {
		img, _ = s.Attr("src")
		img = "https://ya62.ru" + img
	})

	doc.Find("div.news-detail p").Each(func(i int, s *goquery.Selection) {
		txt += s.Text()
		txt = space.ReplaceAllString(txt, " ")
		txt = all.ReplaceAllString(txt, " ")
		txt = strings.Replace(txt, "<...>", "", 3)
		txt = strings.Replace(txt, "YA62.ru", "", 3)
		txt = strings.TrimSpace(txt)
	})

	post.Hash = core.stringToHash(title)
	post.Title = title
	post.Body = txt
	post.Image = img
	post.Link = link

	return post, nil
}
