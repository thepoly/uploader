package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
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
	// cache for caching results of expensive method calls
	m     sync.Mutex
	cache map[string]interface{}
}

type IDMLLink struct {
	ResourceURI string `xml:"LinkResourceURI,attr"`
}

func (s *Story) cacheGet(key string) (interface{}, bool) {
	s.m.Lock()
	val, ok := s.cache[key]
	s.m.Unlock()
	return val, ok
}

func (s *Story) cacheSet(key string, val interface{}) {
	s.m.Lock()
	s.cache[key] = val
	s.m.Unlock()
}

func (s *Story) CreateWPPost() WPPost {
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

func (s *Story) AuthorName() string {
	if val, ok := s.cacheGet("AuthorName"); ok {
		return val.(string)
	}
	for _, story := range s.IDMLStories {
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

func (s *Story) AuthorTitle() string {
	if val, ok := s.cacheGet("AuthorTitle"); ok {
		return val.(string)
	}
	for _, story := range s.IDMLStories {
		for _, paragraph := range story.IDMLParagraphStyleRanges {
			style := paragraph.AppliedParagraphStyle
			if style == "ParagraphStyle/Author Job" {
				res := paragraph.IDMLCharacterStyleRanges[0].Content[0]
				s.cacheSet("AuthorTitle", res)
				return res
			}
		}
	}
	return ""
}

func (s Story) Kicker() string {
	if val, ok := s.cacheGet("Kicker"); ok {
		return val.(string)
	}
	for _, story := range s.IDMLStories {
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

func (s Story) BodyText() string {
	if val, ok := s.cache["BodyText"]; ok {
		return val.(string)
	}
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
				s.cache["BodyText"] = bodyText
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

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("drive-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func (s Story) Photo() []byte {
	if val, ok := s.cache["Photo"]; ok {
		return val.([]byte)
	}

	if len(s.IDMLLinks) == 0 {
		return []byte("")
	}
	// this only grabs the first one...
	uri := s.IDMLLinks[0].ResourceURI

	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/drive-go-quickstart.json
	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}
	unescaped, err := url.PathUnescape(uri)
	if err != nil {
		log.Fatal(err)
	}

	toFind := "/Team Drives/The Polytechnic/"
	idx := strings.Index(unescaped, toFind)
	path := unescaped[idx+len(toFind):]
	parent := "0ACukZyn2MrvEUk9PVA"
	nextLevel := path[:strings.Index(path, "/")]
	filename := ""

	done := false
	for !done {
		var q string
		if nextLevel != "" {
			q = fmt.Sprintf("mimeType = 'application/vnd.google-apps.folder' and '%s' in parents", parent)
			// log.Println("Looking for", nextLevel)
		} else {
			idx := strings.Index(path, "/")
			filename = path[idx+1:]
			q = fmt.Sprintf("name = '%s' and '%s' in parents", filename, parent)
			// log.Println("Looking for", filename)
		}

		r, err := srv.Files.List().PageSize(10).Q(q).
			Fields("nextPageToken, files(id, name)").
			SupportsTeamDrives(true).IncludeTeamDriveItems(true).
			TeamDriveId("0ACukZyn2MrvEUk9PVA").Corpora("teamDrive").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
		}
		if len(r.Files) > 0 {
			found := false
			for _, i := range r.Files {
				// fmt.Printf("%s (%s)\n", i.Name, i.Id)
				if i.Name == nextLevel {
					found = true
					idx := strings.Index(path, "/")
					path = path[idx+1:]
					idx = strings.Index(path, "/")
					if idx == -1 {
						nextLevel = ""
						parent = i.Id
						break
					}
					nextLevel = path[:idx]
					parent = i.Id
					break
				} else if i.Name == filename {
					// Download the file
					resp, err := srv.Files.Get(i.Id).Download()
					if err != nil {
						log.Fatal(err)
					}
					data, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Fatal(err)
					}
					s.cache["Photo"] = data
					return data
					// found = true
					// done = true
					// break
				}
			}
			if !found {
				done = true
			}
		} else {
			fmt.Println("No files found.")
		}
	}

	return []byte("")
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
	if photoByline != "" && len(photo) == 0 {
		validationErrors = append(validationErrors, "Photo byline without photo.")
	}
	if photoCaption != "" && len(photo) == 0 {
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
	photo := s.Photo()
	if len(photo) > 0 {
		fmt.Printf("%13s: %.2f MB\n", "Photo", float64(len(s.Photo()))/1024/1024)
	} else {
		fmt.Printf("%13s:\n", "Photo")
	}
	fmt.Printf("%13s: %s\n", "Photo byline", s.PhotoByline())
	fmt.Printf("%13s: %.80s...\n", "Photo caption", s.PhotoCaption())
	fmt.Printf("%13s: %.80s...\n", "Body text", s.BodyText())
}

// func (s *Story) MarshalJSON ([]byte, error) {
//     storyJSON := bytes.NewBufferString("{")
// }

func NewStory() Story {
	return Story{
		IDMLStories: []IDMLStory{},
		IDMLLinks:   []IDMLLink{},
		cache:       make(map[string]interface{}),
	}
}

func NewStoryFromFile(f io.Reader) Story {
	story := NewStory()
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
				story.IDMLStories = append(story.IDMLStories, idmlStory)
			case "Link":
				idmlLink := IDMLLink{}
				decoder.DecodeElement(&idmlLink, &se)
				story.IDMLLinks = append(story.IDMLLinks, idmlLink)
			}
		}
	}
	return story
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

	story := NewStory()
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
