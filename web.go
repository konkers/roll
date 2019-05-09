package roll

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	rpcjson "github.com/gorilla/rpc/v2/json"
)

type Duration struct {
	time.Duration
}
type Time struct {
	time.Time
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var err error
	d.Duration, err = time.ParseDuration(v)
	if err != nil {
		return err
	}
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(time.RFC1123))
}

func (t *Time) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var err error
	t.Time, err = time.Parse(time.RFC1123, v)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) getTemplate(filename string) (*template.Template, error) {
	file, err := b.openFile(path.Join("templates", filename))
	if err != nil {
		return nil, fmt.Errorf("Can't find template %s: %v", filename, err)
	}
	d, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Error reading template %s: %v", filename, err)
	}

	t := template.Must(
		template.New(filename).
			Funcs(b.funcMap).
			Parse(string(d)))

	return t, nil
}

func (b *Bot) AddTemplateFunc(name string, f interface{}) error {
	if _, ok := b.funcMap[name]; ok {
		return fmt.Errorf("%s template func already registered", name)
	}
	b.funcMap[name] = f
	return nil
}

func (b *Bot) execTemplate(templatePath string, w http.ResponseWriter, req *http.Request) {
	subject, ok := req.Context().Value("subject").(string)
	log.Printf("subject: %v %s", ok, subject)
	t, err := b.getTemplate(templatePath)
	if err != nil {
		log.Printf("Can't get template %s: %v", templatePath, err)
		// do 404
		return
	}

	err = t.Execute(w, b)
	if err != nil {
		log.Printf("Can't execute template %s: %v", templatePath, err)
		// do error
		return
	}
}

func (b *Bot) indexHandler(w http.ResponseWriter, req *http.Request) {
	b.execTemplate("index.html", w, req)
}

func (b *Bot) redirectHandler(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	addr := b.Config.HTTPRedirectBase
	if addr == "" {
		addr = b.Config.HTTPSAddr
	}
	target := "https://" + addr + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	log.Printf("redirect to: %s", target)
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}

func (b *Bot) urlFunc(uri string) template.HTML {
	return template.HTML(fmt.Sprintf("http://%s/%s", b.Config.HTTPSAddr, uri))
}

func (b *Bot) startWebserver() error {
	b.AddTemplateFunc("url", b.urlFunc)

	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.Handle("/auth/user", b.authMiddleware(http.HandlerFunc(b.authUserHandler)))
	r.HandleFunc("/auth", b.authHandler)
	r.HandleFunc("/wiki/{page}", b.wikiHandler)
	r.Handle("/", b.authMiddleware(http.HandlerFunc(b.indexHandler)))

	s := rpc.NewServer()
	s.RegisterCodec(rpcjson.NewCodec(), "application/json")

	for name, mod := range b.modules {
		if provider, ok := mod.(RPCServiceProvider); ok {
			s.RegisterService(provider.GetRPCService(), name)
		}
	}
	r.Handle("/rpc", b.authMiddleware(s))

	cert, err := tls.LoadX509KeyPair(b.Config.CertFile, b.Config.KeyFile)
	if err != nil {
		return err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	log.Printf("About to listen on https://%s/", b.Config.HTTPSAddr)
	listener, err := tls.Listen("tcp", b.Config.HTTPSAddr, config)
	if err != nil {
		return err
	}

	go http.Serve(listener, r)

	if b.Config.HTTPAddr != "" {
		log.Printf("Starting HTTP redirector on http://%s/", b.Config.HTTPAddr)
		listener, err := net.Listen("tcp", b.Config.HTTPAddr)
		if err != nil {
			return err
		}

		go http.Serve(listener, http.HandlerFunc(b.redirectHandler))
	}

	return nil
}
