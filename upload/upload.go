package upload

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

const APIRoot = "https://poly.rpi.edu/wp-json"

type WPPostReturned struct {
	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Meta struct {
		AuthorName string
		Kicker     string
	} `json:"meta"`
	Link string
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
	IDMLLinks   []IDMLLink
}

type IDMLLink struct {
	ResourceURI string `xml:"LinkResourceURI,attr"`
}

func (s Story) CreateWPPost() WPPost {
	wpPost := WPPost{}
	wpPost.Title = s.Headline()
	wpPost.Status = "future"

	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	wpPost.Date = date

	wpPost.Meta.AuthorName = s.AuthorName()
	wpPost.Meta.AuthorTitle = s.AuthorTitle()
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

func (s Story) AuthorTitle() string {
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

func (s Story) PhotoByline() string {
	for _, story := range s.IDMLStories {
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

func (s Story) PhotoCaption() string {
	for _, story := range s.IDMLStories {
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

func (s Story) Photo() string {
	// this only grabs the first one...
	for _, link := range s.IDMLLinks {
		return link.ResourceURI
	}
	return ""
}

// Validate checks some things that should be consistent in all articles, e.g.
// has a headline and author, no double spaces, there's a photo if there's a caption, etc.
// Errors found here are usually the result of making an improper snippet in InDesign.
// Any failure here will prevent the article from being posted to the website.
// We may want to add a command line flag in the future to ignore validation errors.
func (s Story) Validate() []string {
	validationErrors := []string{}

	headline := s.Headline()
	if headline == "" {
		validationErrors = append(validationErrors, "No headline.")
	}
	if strings.Contains(headline, "  ") {
		validationErrors = append(validationErrors, "Headline contains two consecutive spaces.")
	}

	authorName := s.AuthorName()
	if authorName == "" {
		validationErrors = append(validationErrors, "No author name.")
	}
	if strings.Contains(authorName, "  ") {
		validationErrors = append(validationErrors, "Author name contains two consecutive spaces.")
	}

	kicker := s.Kicker()
	if kicker == "" {
		validationErrors = append(validationErrors, "No kicker.")
	}
	if strings.Contains(kicker, "  ") {
		validationErrors = append(validationErrors, "Kicker contains two consecutive spaces.")
	}

	authorTitle := s.AuthorTitle()
	if authorTitle == "" {
		validationErrors = append(validationErrors, "No author title.")
	}
	if strings.Contains(authorTitle, "  ") {
		validationErrors = append(validationErrors, "Author title contains two consecutive spaces.")
	}

	bodyText := s.BodyText()
	if bodyText == "" {
		validationErrors = append(validationErrors, "No body text.")
	} else if len(bodyText) < 100 {
		msg := fmt.Sprintf("Body text extremely short (%d characters).", len(bodyText))
		validationErrors = append(validationErrors, msg)
	}

	photo := s.Photo()
	photoByline := s.PhotoByline()
	photoCaption := s.PhotoCaption()
	if photoByline != "" && photo == "" {
		validationErrors = append(validationErrors, "Photo byline without photo.")
	}
	if photoCaption != "" && photo == "" {
		validationErrors = append(validationErrors, "Photo caption without photo.")
	}

	if strings.Contains(photoByline, "  ") {
		validationErrors = append(validationErrors, "Photo byline contains two consecutive spaces.")
	}
	if strings.Contains(photoCaption, "  ") {
		validationErrors = append(validationErrors, "Photo caption contains two consecutive spaces.")
	}

	return validationErrors
}

func (s Story) Print() {
	fmt.Printf("%15s\n", "Story")
	fmt.Printf("-------------------------\n")
	fmt.Printf("%13s: %s\n", "Kicker", s.Kicker())
	fmt.Printf("%13s: %s\n", "Headline", s.Headline())
	fmt.Printf("%13s: %s\n", "Author name", s.AuthorName())
	fmt.Printf("%13s: %s\n", "Author title", s.AuthorTitle())
	fmt.Printf("%13s: %s\n", "Photo", s.Photo())
	fmt.Printf("%13s: %s\n", "Photo byline", s.PhotoByline())
	fmt.Printf("%13s: %.80s...\n", "Photo caption", s.PhotoCaption())
	fmt.Printf("%13s: %.80s...\n", "Body text", s.BodyText())
}

func ParseAndUpload(apiPassword, snippetPath string) {
	c := color.New(color.FgCyan)
	c.Printf("Reading \"%s\"...", snippetPath)
	file, err := os.Open(snippetPath)
	if err != nil {
		fmt.Println()
		r := color.New(color.FgRed)
		r.Println(err)
		return
	}

	story := Story{IDMLStories: []IDMLStory{}}
	decoder := xml.NewDecoder(file)
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
				story.IDMLStories = append(story.IDMLStories, idmlStory)
			case "Link":
				idmlLink := IDMLLink{}
				decoder.DecodeElement(&idmlLink, &se)
				story.IDMLLinks = append(story.IDMLLinks, idmlLink)
			}
		}
	}
	c.Printf(" done.\n")

	validationErrors := story.Validate()
	if len(validationErrors) > 0 {
		color.Red("Validation errors.")
	} else {
		color.Green("✓ Validation succeeded.")
	}
	for _, validationError := range validationErrors {
		color.Yellow("\t● " + validationError)
	}
	if len(validationErrors) > 0 {
		color.Red("Aborting.")
		return
	}

	fmt.Println()
	story.Print()
	fmt.Println()

	client := http.Client{Timeout: 10 * time.Second}
	body, err := json.Marshal(story.CreateWPPost())
	if err != nil {
		r := color.New(color.FgRed)
		r.Println(err)
		log.Println(err)
		return
	}

	// check if we already uploaded this
	req, err := http.NewRequest("GET", APIRoot+"/wp/v2/posts?per_page=30&status=any", nil)
	req.SetBasicAuth("uploader", apiPassword)
	if err != nil {
		r := color.New(color.FgRed)
		r.Println(err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		r := color.New(color.FgRed)
		r.Println(err)
		return
	}
	if resp.StatusCode == 400 {
		data, _ := ioutil.ReadAll(resp.Body)
		color.Red("WordPress communication failed: %s", string(data))
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		r := color.New(color.FgRed)
		r.Println(err)
		return
	}
	posts := []WPPostReturned{}
	err = json.Unmarshal(data, &posts)
	if err != nil {
		r := color.New(color.FgRed)
		r.Println(err)
		return
	}
	for _, post := range posts {
		// See if we have a recent post with the same headline
		// OR the same kicker and author.
		if post.Title.Rendered == story.Headline() ||
			(post.Meta.Kicker == story.Kicker() && post.Meta.AuthorName == story.AuthorName()) {
			color.Yellow("Similar post already exists: %s", post.Link)
			color.Red("Aborting.")
			return
		}
	}

	// create post
	c.Print("Uploading... ")
	req, err = http.NewRequest("POST", APIRoot+"/wp/v2/posts", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println()
		r := color.New(color.FgRed)
		r.Println(err)
		return
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("uploader", apiPassword)
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println()
		r := color.New(color.FgRed)
		r.Println(err)
		return
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println()
		r := color.New(color.FgRed)
		r.Println(err)
		return
	}
	g := color.New(color.FgGreen)
	g.Println(" done. ✓")
}
