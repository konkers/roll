package roll

import (
	"html/template"
	"log"
	"net/http"
)

func (b *Bot) indexHandler(w http.ResponseWriter, req *http.Request) {
	var indexTemplate = template.Must(template.ParseFiles("templates/index.html"))
	indexTemplate.Execute(w, b)
	//w.Header().Set("Content-Type", "text/html")
	//w.Write([]byte("<html><body><a href=\"https://id.twitch.tv/oauth2/authorize?client_id=<CLIENTID>&redirect_uri=https://localhost:10443/auth&response_type=token&scope=channel_editor+channel_subscriptions+channel_read\">Authorize</a></body></html>"))
}

func (b *Bot) redirectHandler(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	target := "https://" + b.config.HTTPRedirectBase + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	log.Printf("redirect to: %s", target)
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}

func (b *Bot) startWebserver() error {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", b.indexHandler)

	log.Printf("About to listen on https://%s/", b.config.HTTPSAddr)
	go func() {
		err := http.ListenAndServeTLS(b.config.HTTPSAddr, b.config.CertFile, b.config.KeyFile, nil)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}()

	if b.config.HTTPAddr != "" {
		log.Printf("Starting HTTP redirecter on http://%s/", b.config.HTTPAddr)
		go http.ListenAndServe(b.config.HTTPAddr, http.HandlerFunc(b.redirectHandler))
	}

	return nil
}
