package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"

	"github.com/thepoly/uploader/story"
)

type Server struct {
	listenAddr    string
	handler       http.Handler
	wpAPIPassword string
	storyManager  *story.Manager
}

func New(apiPassword string) (*Server, error) {
	sm, err := story.NewManager()
	if err != nil {
		return nil, err
	}

	server := &Server{
		listenAddr:    "127.0.0.1:8000",
		wpAPIPassword: apiPassword,
		storyManager:  sm,
	}

	router := chi.NewRouter()
	cors := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	router.Use(cors.Handler)
	router.Post("/validate-story", server.ValidateStoryHandler)
	router.Get("/available-stories", server.GetAvailableStories)
	server.handler = router

	return server, nil
}

func (s *Server) Run() {
	log.Println("Server listening on", s.listenAddr)
	http.ListenAndServe(s.listenAddr, s.handler)
}

// func (s *Server) ValidateSnippetHandler(w http.ResponseWriter, req *http.Request) {
// 	log.Println("new snippet!")
// 	story := upload.NewStoryFromFile(req.Body)
// 	w.Header().Set("Content-Type", "application/json")
// 	encoder := json.NewEncoder(w)
// 	err := encoder.Encode(&story)
// 	if err != nil {
// 		http.Error(w, "Unable to marshal story", 500)
// 		return
// 	}
// }

func (s *Server) GetAvailableStories(w http.ResponseWriter, req *http.Request) {
	stories := s.storyManager.GetStories()
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(&stories)
	if err != nil {
		http.Error(w, "Unable to marshal available stories", 500)
		return
	}
}

func (s *Server) ValidateStoryHandler(w http.ResponseWriter, req *http.Request) {
	story := &story.Story{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(story)
	if err != nil {
		http.Error(w, "Unable to decode story", 500)
		return
	}
	validationErrors := story.ValidationErrors()
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(&validationErrors)
	if err != nil {
		http.Error(w, "Unable to encode validation errors", 500)
		return
	}
}
