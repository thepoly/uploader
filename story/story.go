package story

import (
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

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
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

type Story struct {
	Snippet     *Snippet `json:"snippet"`
	Headline    string   `json:"headline"`
	Kicker      string   `json:"kicker"`
	AuthorName  string   `json:"authorName"`
	AuthorTitle string   `json:"authorTitle"`
	BodyText    string   `json:"bodyText"`
	Subdeck     string   `json:"subdeck"`
}

type IDMLLink struct {
	ResourceURI string `xml:"LinkResourceURI,attr"`
}

type Manager struct {
	driveClient      *drive.Service
	m                sync.Mutex
	availableStories []*Story
}

func NewManager() (*Manager, error) {
	ctx := context.Background()
	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
		return nil, err
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/drive-go-quickstart.json
	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
		return nil, err
	}
	client := getClient(ctx, config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
		return nil, err
	}
	m := &Manager{
		driveClient: srv,
	}

	go m.updater()

	return m, nil
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

func (m *Manager) updater() {
	m.update()
	for range time.Tick(time.Second * 10) {
		// m.update()
	}
}

func (m *Manager) update() {
	var q string
	when := time.Now().Add(time.Hour * 24 * -5).Format(time.RFC3339)
	q = fmt.Sprintf("name contains '.idms' and mimeType = 'text/xml' and modifiedTime >= '%s'", when)

	snippets := []*Snippet{}
	r, err := m.driveClient.Files.List().PageSize(10).Q(q).
		Fields("nextPageToken, files(id, name, modifiedTime, mimeType)").
		SupportsTeamDrives(true).IncludeTeamDriveItems(true).
		TeamDriveId("0ACukZyn2MrvEUk9PVA").Corpora("teamDrive").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
		return
	}

	for _, i := range r.Files {
		snippet := NewSnippet()
		snippet.Name = i.Name
		snippet.DriveID = i.Id
		modifiedTime, err := time.Parse(time.RFC3339, i.ModifiedTime)
		if err != nil {
			log.Fatal(err)
		}
		snippet.LastModified = modifiedTime
		resp, err := m.driveClient.Files.Get(i.Id).Download()
		if err != nil {
			log.Fatal(err)
		}

		snippet.ParseFile(resp.Body)
		snippets = append(snippets, &snippet)
	}

	stories := []*Story{}
	for _, snippet := range snippets {
		story := &Story{
			Snippet:     snippet,
			Headline:    snippet.Headline(),
			Kicker:      snippet.Kicker(),
			AuthorName:  snippet.AuthorName(),
			AuthorTitle: snippet.AuthorTitle(),
			BodyText:    snippet.BodyText(),
		}
		stories = append(stories, story)
	}

	m.m.Lock()
	m.availableStories = stories
	m.m.Unlock()
}

func (m *Manager) GetStories() []*Story {
	stories := []*Story{}
	m.m.Lock()
	for _, story := range m.availableStories {
		stories = append(stories, story)
	}
	m.m.Unlock()
	return stories
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
				res := paragraph.IDMLCharacterStyleRanges[0].Content[0]
				s.cacheSet("AuthorTitle", res)
				return res
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
				if strings.Contains(bodyText, "Etzine") {
					log.Println(bodyText)
				}
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
