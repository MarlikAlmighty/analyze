package app

import (
	"errors"
	"github.com/MarlikAlmighty/analyze-it/internal/models"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
)

func (core *Core) getLinkRzn(html string) (map[string]string, error) {

	var (
		doc *goquery.Document
		err error
	)

	if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
		return nil, err
	}

	m := make(map[string]string)
	doc.Find("#news-container > .stories .stories-item__title > a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		title := s.Text()
		m[link] = title
	})

	if len(m) == 0 {
		return nil, errors.New("links is zero")
	}

	return m, nil
}

func (core *Core) catchPostFromRzn(html string) (models.Post, error) {

	reg := regexp.MustCompile(`\s+`)

	var (
		title, link, img string
		err              error
	)

	post := models.Post{}

	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
		return post, errors.New(err.Error())
	}

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
	post.Hash = core.stringToHash(title)
	post.Title = title
	post.Body = body
	post.Image = img
	post.Link = link

	return post, nil
}
