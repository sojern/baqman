package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

const (
	sessionName = "killedjobs"
)

var (
	logger = log.New(os.Stdout, "[baqman] ", log.Ldate|log.Ltime)
)

// Server holds a value for BQService and page templates
type Server struct {
	bqService *BQService
	templates map[string]*template.Template
	store     *sessions.CookieStore
}

// RunServer starts a new server for BaqMan on provided port
func RunServer(service *BQService, port int, sessionSecret string) {

	s := &Server{
		bqService: service,
		templates: map[string]*template.Template{},
		store:     sessions.NewCookieStore([]byte(sessionSecret)),
	}

	s.templates["completed"] = template.Must(template.ParseFiles("web/html/completed.html", "web/html/base.html"))
	s.templates["index"] = template.Must(template.ParseFiles("web/html/index.html", "web/html/base.html"))
	s.templates["describe"] = template.Must(template.ParseFiles("web/html/describe.html", "web/html/base.html"))

	router := mux.NewRouter().StrictSlash(true)
	assetsHandler := http.FileServer(http.Dir("web/"))

	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", assetsHandler))
	router.HandleFunc("/", s.index)
	router.HandleFunc("/completed", s.completed)
	router.HandleFunc("/describe/{JobID}", s.describe).Methods("GET")
	router.HandleFunc("/kill/{JobID}", s.killJob).Methods("GET")
	router.HandleFunc("/killmany", s.killMany).Methods("POST")
	router.HandleFunc("/_ah/health", s.healthCheckHandler)

	httpserver := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handlers.CombinedLoggingHandler(os.Stdout, router),
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 40 * time.Second,
	}

	httpserver.ListenAndServe()
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	flashes := []string{}
	if session, err := s.store.Get(r, sessionName); err == nil {
		for _, f := range session.Flashes() {
			flashes = append(flashes, f.(string))
		}
		session.Save(r, w)
	}

	jobs := s.bqService.GetJobs("")
	if err := s.templates["index"].ExecuteTemplate(w, "base", struct {
		Flashes []string
		Jobs    *Jobs
	}{flashes, jobs}); err != nil {
		log.Println(err)
	}
}

func (s *Server) completed(w http.ResponseWriter, r *http.Request) {
	pageToken := r.URL.Query().Get("token")
	jobs := s.bqService.GetJobs(pageToken)
	if err := s.templates["completed"].ExecuteTemplate(w, "base", struct {
		Flashes []string
		Jobs    *Jobs
	}{nil, jobs}); err != nil {
		log.Println(err)
	}
}

func (s *Server) describe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["JobID"]

	flashes := []string{}
	if session, err := s.store.Get(r, sessionName); err == nil {
		for _, f := range session.Flashes() {
			flashes = append(flashes, f.(string))
		}
		session.Save(r, w)
	}

	job := s.bqService.GetJob(jobID)
	if err := s.templates["describe"].ExecuteTemplate(w, "base", struct {
		Flashes []string
		Job     *Job
	}{flashes, job}); err != nil {
		log.Println(err)
	}
}

func (s *Server) killMany(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	jobsToKill := r.Form["jobkill"]
	if session, err := s.store.Get(r, sessionName); err != nil {
		log.Printf("error fetching session %v", err)
	} else {
		for _, jobID := range jobsToKill {
			log.Printf("cancelling job %v", jobID)
			s.bqService.CancelJob(jobID)
			session.AddFlash(jobID)
		}
		session.Save(r, w)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) killJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["JobID"]

	if session, err := s.store.Get(r, sessionName); err != nil {
		log.Printf("error fetching session %v", err)
	} else {
		session.AddFlash(jobID)
		session.Save(r, w)
	}

	log.Printf("cancelling job %v", jobID)
	s.bqService.CancelJob(jobID)
	http.Redirect(w, r, fmt.Sprintf("/describe/%s", jobID), http.StatusSeeOther)
}

func (s *Server) healthCheckHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "ok")
}
