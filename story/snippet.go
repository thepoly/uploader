package story

import (
	"encoding/xml"
	"io"
	"strings"
	"sync"
	"time"
)

type Snippet struct {
	Name         string    `json:"name"`
	DriveID      string    `json:"driveID"`
	LastModified time.Time `json:"lastModified"`
	idmlStories  []IDMLStory
	idmlLinks    []IDMLLink
	// cache for caching results of expensive method calls
	m     sync.Mutex
	cache map[string]interface{}
}

type IDMLStory struct {
	IDMLParagraphStyleRanges []IDMLParagraphStyleRange `xml:"ParagraphStyleRange"`
}

type IDMLParagraphStyleRange struct {
	IDMLCharacterStyleRanges []IDMLCharacterStyleRange `xml:"CharacterStyleRange"`
	AppliedParagraphStyle    string                    `xml:",attr"`
}

type IDMLCharacterStyleRange struct {
	Content   []string
	FontStyle string `xml:",attr"`
}

// func (s *Snippet) CreateWPPost() WPPost {
// 	wpPost := WPPost{}
// 	wpPost.Title = s.Headline()
// 	wpPost.Status = "future"
//
// 	now := time.Now()
// 	date := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
// 	wpPost.Date = date
//
// 	wpPost.Meta.AuthorName = s.AuthorName()
// 	wpPost.Meta.AuthorTitle = s.AuthorTitle()
// 	wpPost.Meta.Kicker = s.Kicker()
// 	wpPost.Content = s.BodyText()
// 	return wpPost
// }

func NewSnippet() Snippet {
	return Snippet{
		idmlStories: []IDMLStory{},
		idmlLinks:   []IDMLLink{},
		cache:       make(map[string]interface{}),
	}
}

func (s *Snippet) cacheGet(key string) (interface{}, bool) {
	s.m.Lock()
	val, ok := s.cache[key]
	s.m.Unlock()
	return val, ok
}

func (s *Snippet) cacheSet(key string, val interface{}) {
	s.m.Lock()
	s.cache[key] = val
	s.m.Unlock()
}

func (s *Snippet) ParseFile(f io.Reader) {
	decoder := xml.NewDecoder(f)
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "Story":
				idmlStory := IDMLStory{}
				decoder.DecodeElement(&idmlStory, &se)
				s.idmlStories = append(s.idmlStories, idmlStory)
			case "Link":
				idmlLink := IDMLLink{}
				decoder.DecodeElement(&idmlLink, &se)
				s.idmlLinks = append(s.idmlLinks, idmlLink)
			}
		}
	}
}

func (s *Snippet) AuthorName() string {
	if val, ok := s.cacheGet("AuthorName"); ok {
		return val.(string)
	}
	for _, story := range s.idmlStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Author" {
				res := paragraph.IDMLCharacterStyleRanges[0].Content[0]
				s.cacheSet("AuthorName", res)
				return res
			}
		}
	}
	return ""
}

func (s *Snippet) AuthorTitle() string {
	if val, ok := s.cacheGet("AuthorTitle"); ok {
		return val.(string)
	}
	for _, story := range s.idmlStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Author Job" {
				authorTitle := ""
				for _, characterRange := range paragraph.IDMLCharacterStyleRanges {
					// look for regular because author title line is italicized by default
					if characterRange.FontStyle == "Regular" {
						authorTitle += "<i>"
					}
					for _, content := range characterRange.Content {
						authorTitle += content
					}
					if characterRange.FontStyle == "Regular" {
						authorTitle += "</i>"
					}
				}
				s.cache["AuthorTitle"] = authorTitle
				return authorTitle
			}
		}
	}
	return ""
}

func (s *Snippet) Kicker() string {
	if val, ok := s.cacheGet("Kicker"); ok {
		return val.(string)
	}
	for _, story := range s.idmlStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Kicker" {
				res := paragraph.IDMLCharacterStyleRanges[0].Content[0]
				s.cacheSet("Kicker", res)
				return res
			}
		}
	}
	return ""
}

func (s *Snippet) BodyText() string {
	if val, ok := s.cache["BodyText"]; ok {
		return val.(string)
	}
	for _, story := range s.idmlStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Body Text" {
				bodyText := "<p>"
				for _, characterRange := range paragraph.IDMLCharacterStyleRanges {
					if characterRange.FontStyle == "Italic" {
						bodyText += "<i>"
					}
					for _, content := range characterRange.Content {
						for _, char := range content {
							if char == '\t' {
								bodyText += "</p><p>"
							} else {
								bodyText += string(char)
							}
						}
					}
					if characterRange.FontStyle == "Italic" {
						bodyText += "</i>"
					}
				}
				bodyText += "</p>"
				s.cache["BodyText"] = bodyText
				return bodyText
			}
		}
	}
	return ""
}

func (s *Snippet) Headline() string {
	for _, story := range s.idmlStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if strings.Contains(style, "Headline") {
				headline := ""
				for _, characterRange := range paragraph.IDMLCharacterStyleRanges {
					for _, content := range characterRange.Content {
						headline += content
					}
				}
				return headline
			}
		}
	}
	return ""
}

func (s *Snippet) PhotoByline() string {
	for _, story := range s.idmlStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Photo Byline" {
				photoByline := ""
				for _, characterRange := range paragraph.IDMLCharacterStyleRanges {
					for _, content := range characterRange.Content {
						photoByline += content
					}
				}
				return photoByline
			}
		}
	}
	return ""
}

func (s *Snippet) PhotoCaption() string {
	for _, story := range s.idmlStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Caption" {
				caption := ""
				for _, characterRange := range paragraph.IDMLCharacterStyleRanges {
					for _, content := range characterRange.Content {
						caption += content
					}
				}
				return caption
			}
		}
	}
	return ""
}
