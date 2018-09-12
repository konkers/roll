package roll

import (
	"html/template"
	"log"
	"net/http"
)

func (b *Bot) authHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("auth")
	var authTemplate = template.Must(template.ParseFiles("templates/auth.html"))
	err := authTemplate.Execute(w, b)
	if err != nil {
		log.Println(err)
	}
}
