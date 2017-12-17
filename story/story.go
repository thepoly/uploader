package story

import (
	"context"
	"encoding/json"
	"fmt"
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

func (s *Story) ValidationErrors() []string {
	validationErrors := []string{}

	if s.Headline == "" {
		validationErrors = append(validationErrors, "No headline.")
	}
	if strings.Contains(s.Headline, "  ") {
		validationErrors = append(validationErrors, "Headline contains two consecutive spaces.")
	}

	if s.AuthorName == "" {
		validationErrors = append(validationErrors, "No author name.")
	}
	if strings.Contains(s.AuthorName, "  ") {
		validationErrors = append(validationErrors, "Author name contains two consecutive spaces.")
	}

	if s.Kicker == "" {
		validationErrors = append(validationErrors, "No kicker.")
	}
	if strings.Contains(s.Kicker, "  ") {
		validationErrors = append(validationErrors, "Kicker contains two consecutive spaces.")
	}

	if s.AuthorTitle == "" {
		validationErrors = append(validationErrors, "No author title.")
	}
	if strings.Contains(s.AuthorTitle, "  ") {
		validationErrors = append(validationErrors, "Author title contains two consecutive spaces.")
	}

	if s.BodyText == "" {
		validationErrors = append(validationErrors, "No body text.")
	} else if len(s.BodyText) < 100 {
		msg := fmt.Sprintf("Body text extremely short (%d characters).", len(s.BodyText))
		validationErrors = append(validationErrors, msg)
	}

	// photo := s.Photo()
	// photoByline := s.PhotoByline()
	// photoCaption := s.PhotoCaption()
	// if photoByline != "" && len(photo) == 0 {
	// 	validationErrors = append(validationErrors, "Photo byline without photo.")
	// }
	// if photoCaption != "" && len(photo) == 0 {
	// 	validationErrors = append(validationErrors, "Photo caption without photo.")
	// }
	//
	// if strings.Contains(photoByline, "  ") {
	// 	validationErrors = append(validationErrors, "Photo byline contains two consecutive spaces.")
	// }
	// if strings.Contains(photoCaption, "  ") {
	// 	validationErrors = append(validationErrors, "Photo caption contains two consecutive spaces.")
	// }

	return validationErrors
}
