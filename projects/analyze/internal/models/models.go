package models

type Array []*Post

type Post struct {
	Title string `json:"Title,omitempty"`
	Body  string `json:"Body,omitempty"`
	Link  string `json:"Link,omitempty"`
	Image string `json:"Image,omitempty"`
	Hash  string `json:"Hash,omitempty"`
}
