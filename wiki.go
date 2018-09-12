package roll

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gomarkdown/markdown"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/gorilla/mux"
)

var ()

func (b *Bot) wikiHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	page, ok := vars["page"]
	if !ok {
		log.Println("No page variable")
		http.NotFound(w, req)
		return
	}

	filename := path.Join("wiki", page+".md")
	f, err := os.Open(filename)
	if err != nil {
		log.Printf("Can't open: %s", filename)
		http.NotFound(w, req)
		return
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Printf("Can't read: %s", filename)
		http.NotFound(w, req)
		return
	}

	title := ""
	titleHook := func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
		// Filter out Title blocks (ex: %Title) and save the title.
		if h, ok := node.(*ast.Heading); ok {
			for _, n := range h.GetChildren() {
				if t, ok := n.(*ast.Text); ok && h.IsTitleblock {
					title = string(t.Literal)
					return ast.SkipChildren, true
				}
			}
		}
		return ast.GoToNext, false
	}

	mdExtensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.Titleblock
	mdParser := parser.NewWithExtensions(mdExtensions)
	opts := html.RendererOptions{
		Flags:          html.CommonFlags,
		RenderNodeHook: titleHook,
	}
	renderer := html.NewRenderer(opts)
	output := markdown.ToHTML(data, mdParser, renderer)

	templateData := struct {
		Title string
		Body  template.HTML
	}{
		title,
		template.HTML(output),
	}

	var wikiTemplate = template.Must(template.ParseFiles("templates/wiki.html"))
	wikiTemplate.Execute(w, templateData)
}
