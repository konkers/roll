package roll

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (b *Bot) indexHandler(w http.ResponseWriter, req *http.Request) {
	var indexTemplate = template.Must(template.ParseFiles("templates/index.html"))
	indexTemplate.Execute(w, b)
}

func (b *Bot) redirectHandler(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	target := "https://" + b.Config.HTTPRedirectBase + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	log.Printf("redirect to: %s", target)
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}

func (b *Bot) startWebserver() error {
	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.HandleFunc("/auth/", b.authHandler)
	r.HandleFunc("/wiki/{page}", b.wikiHandler)
	r.HandleFunc("/", b.indexHandler)

	log.Printf("About to listen on https://%s/", b.Config.HTTPSAddr)
	go func() {
		err := http.ListenAndServeTLS(b.Config.HTTPSAddr, b.Config.CertFile, b.Config.KeyFile, r)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}()

	if b.Config.HTTPAddr != "" {
		log.Printf("Starting HTTP redirector on http://%s/", b.Config.HTTPAddr)
		go http.ListenAndServe(b.Config.HTTPAddr, http.HandlerFunc(b.redirectHandler))
	}

	return nil
}
