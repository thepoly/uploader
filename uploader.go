package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const APIRoot = "https://poly.rpi.edu/wp-json"

var APIPassword = os.Args[1]
var SnippetPath = os.Args[2]

type WPPostReturned struct {
	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
}

type WPPost struct {
	Title   string     `json:"title"`
	Content string     `json:"content"`
	Meta    WPPostMeta `json:"meta"`
	Status  string     `json:"status"`
	Date    time.Time  `json:"date"`
}

type WPPostMeta struct {
	AuthorName  string `json:"AuthorName"`
	AuthorTitle string `json:"AuthorTitle"`
	Kicker      string `json:"Kicker"`
}

type IDMLStory struct {
	IDMLParagraphStyleRanges []IDMLParagraphStyleRange `xml:"ParagraphStyleRange"`
}

type IDMLParagraphStyleRange struct {
	IDMLCharacterStyleRanges []IDMLCharacterStyleRange `xml:"CharacterStyleRange"`
	AppliedParagraphStyle    string                    `xml:",attr"`
}

type IDMLCharacterStyleRange struct {
	Content []string
}

type Story struct {
	IDMLStories []IDMLStory
}

func (s Story) CreateWPPost() WPPost {
	wpPost := WPPost{}
	wpPost.Title = s.Headline()
	wpPost.Status = "future"

	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	wpPost.Date = date

	wpPost.Meta.AuthorName = s.AuthorName()
	wpPost.Meta.AuthorTitle = s.AuthorJob()
	wpPost.Meta.Kicker = s.Kicker()
	wpPost.Content = s.BodyText()
	return wpPost
}

func (s Story) AuthorName() string {
	for _, story := range s.IDMLStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Author" {
				return paragraph.IDMLCharacterStyleRanges[0].Content[0]
			}
		}
	}
	return ""
}

func (s Story) AuthorJob() string {
	for _, story := range s.IDMLStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Author Job" {
				return paragraph.IDMLCharacterStyleRanges[0].Content[0]
			}
		}
	}
	return ""
}

func (s Story) Kicker() string {
	for _, story := range s.IDMLStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Kicker" {
				return paragraph.IDMLCharacterStyleRanges[0].Content[0]
			}
		}
	}
	return ""
}

func (s Story) BodyText() string {
	for _, story := range s.IDMLStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Body Text" {
				bodyText := ""
				for _, characterRange := range paragraph.IDMLCharacterStyleRanges {
					for _, content := range characterRange.Content {
						for _, char := range content {
							if char == '\t' {
								bodyText += "\n\n"
							} else {
								bodyText += string(char)
							}
						}
					}
				}
				return bodyText
			}
		}
	}
	return ""
}

func (s Story) Headline() string {
	for _, story := range s.IDMLStories {
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

func main() {
	log.Println("parsing")
	file, err := os.Open(SnippetPath)
	if err != nil {
		log.Println(err)
		return
	}

	story := Story{IDMLStories: []IDMLStory{}}
	decoder := xml.NewDecoder(file)
	indent := 0
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			indent++
			indentStr := ""
			for i := 0; i < indent; i++ {
				indentStr += " "
			}
			if se.Name.Local == "Story" {
				// log.Println(indentStr + se.Name.Local)
				idmlStory := IDMLStory{}
				decoder.DecodeElement(&idmlStory, &se)
				story.IDMLStories = append(story.IDMLStories, idmlStory)
			}
		case xml.EndElement:
			indent--
		}
	}
	// log.Println(story.Kicker())
	log.Println(story.Headline() + " by " + story.AuthorName())
	// log.Println(story.AuthorName())
	// log.Println(story.AuthorJob())
	// log.Println(story.BodyText())
	// log.Println(story.CreateWPPost())

	client := http.Client{Timeout: 10 * time.Second}
	body, err := json.Marshal(story.CreateWPPost())
	if err != nil {
		log.Println(err)
		return
	}

	// check if we already uploaded this
	req, err := http.NewRequest("GET", APIRoot+"/wp/v2/posts?per_page=30&status=future", nil)
	req.SetBasicAuth("uploader", APIPassword)
	if err != nil {
		log.Println(err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	if resp.StatusCode == 400 {
		log.Println("unauthorized")
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	posts := []WPPostReturned{}
	err = json.Unmarshal(data, &posts)
	if err != nil {
		log.Println(err)
		return
	}
	for _, post := range posts {
		if post.Title.Rendered == story.Headline() {
			log.Println("future post with same title already exists; aborting")
			return
		}
	}

	// create post
	log.Println("uploading")
	req, err = http.NewRequest("POST", APIRoot+"/wp/v2/posts", bytes.NewBuffer(body))
	if err != nil {
		log.Println(err)
		return
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("uploader", APIPassword)
	resp, err = client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("done")
}
