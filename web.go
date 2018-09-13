package roll

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json"
)

func renderStatus(status *MarathonGameStatus) string {
	if status == nil {
		return "not started"
	}
	switch *status {
	case GameStatusNotStarted:
		return "not started"
	case GameStatusRunning:
		return "running"
	case GameStatusFinished:
		return "finished"
	default:
		return "???"
	}
}

func renderTime(game *MarathonGame) string {
	var d time.Duration
	if game.StartedTime == nil {
		return "?:??:??"
	} else if game.EndedTime == nil {
		d = time.Now().Sub(*game.StartedTime)
	} else {
		d = game.EndedTime.Sub(*game.StartedTime)
	}
	hours := d.Truncate(time.Hour)
	d -= hours
	mins := d.Truncate(time.Minute)
	d -= mins
	seconds := d.Truncate(time.Second)
	return fmt.Sprintf("%01d:%02d:%02d", int(hours.Hours()), int(mins.Minutes()), int(seconds.Seconds()))
}

func (b *Bot) indexHandler(w http.ResponseWriter, req *http.Request) {
	var marathon Marathon
	id := 1
	b.marathon.Get(nil, &id, &marathon)
	var indexTemplate = template.Must(
		template.New("index.html").
			Funcs(template.FuncMap{
				"status": renderStatus,
				"time":   renderTime,
			}).
			ParseFiles("templates/index.html"))
	indexTemplate.Execute(w, marathon)
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

	b.alert = NewAlertService(b)
	b.marathon = NewMarathonService(b)
	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterService(b.alert, "")
	s.RegisterService(b.marathon, "")
	r.Handle("/rpc", s)

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
