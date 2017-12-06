package main

import (
	//	"encoding/xml"
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const APIRoot = "https://poly.rpi.edu/wp-json"

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
						bodyText += content
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

	file, err := os.Open("/Volumes/GoogleDrive/Team Drives/The Polytechnic/Issues/Fall 2017/2017-12-06/Web/features1.idms")
	if err != nil {
		log.Println(err)
		return
	}

	log.Println(file)
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
				log.Println(indentStr + se.Name.Local)
				idmlStory := IDMLStory{}
				decoder.DecodeElement(&idmlStory, &se)
				story.IDMLStories = append(story.IDMLStories, idmlStory)
			}
		case xml.EndElement:
			indent--
		}
	}
	log.Println(story.Kicker())
	log.Println(story.Headline())
	log.Println(story.AuthorName())
	log.Println(story.AuthorJob())
	log.Println(story.BodyText())
	return

	log.Println("uploading")

	client := http.Client{Timeout: 10 * time.Second}

	body := bytes.NewBuffer([]byte(`{
        "content": "test",
        "title": "test",
        "excerpt": "test"
        }`))
	req, err := http.NewRequest("POST", APIRoot+"/wp/v2/posts", body)
	if err != nil {
		log.Println(err)
		return
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("uploader", "haha nice try")
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	strResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(strResp))
}
