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

	dote := regexp.MustCompile(`\.`)

	var (
		post models.Post
		err  error
	)

	var doc *goquery.Document
	if doc, err = goquery.NewDocumentFromReader(strings.NewReader(html)); err != nil {
		return post, errors.New(err.Error())
	}

	doc.Find("#newsContainer > div.row.url-checkpoint.newsItem.story > div.col.story__details > div > div.story__body > div.story__hero > div > img").Each(func(i int, s *goquery.Selection) {
		post.Image, _ = s.Attr("src")
	})

	doc.Find("#newsContainer > div.row.url-checkpoint.newsItem.story").Each(func(i int, s *goquery.Selection) {
		post.Title, _ = s.Attr("data-title")
		post.Link, _ = s.Attr("data-url")
	})

	doc.Find("#newsContainer > div.row.url-checkpoint.newsItem.story > div.col.story__details > div > div.story__body > div:nth-child(3)").Each(func(i int, s *goquery.Selection) {
		post.Body = dote.ReplaceAllString(s.Text(), ". ")
	})

	post.Hash = core.stringToHash(post.Link)

	return post, nil
}
