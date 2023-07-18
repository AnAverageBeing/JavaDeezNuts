package main

import (
	"JavaDeezNuts/utils"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday/v2"
)

const (
	repoRoot      = "web"
	templateDir   = "templates"
	staticDir     = "static"
	logFilePath   = "access_logs.json"
	ipAPIEndpoint = "http://ip-api.com/json/"
)

type Data struct {
	Title   string
	Content template.HTML
}

func renderTemplate(w http.ResponseWriter, tmpl string, data Data) {
	tmplPath := filepath.Join(repoRoot, templateDir, tmpl)
	tmplContent, err := utils.GetFileContent(tmplPath)
	if err != nil {
		http.Error(w, "Error fetching template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmplParsed, err := template.New("").Parse(tmplContent)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmplParsed.Execute(w, data)
	if err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func markdownToHTML(mdContent string) template.HTML {
	htmlContent := blackfriday.Run([]byte(mdContent))
	return template.HTML(htmlContent)
}

func handleStaticFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path[len("/static/"):]
	content, err := utils.GetFileContent(filepath.Join(repoRoot, staticDir, filePath))
	if err != nil {
		http.Error(w, "Error fetching static file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", http.DetectContentType([]byte(content)))

	if strings.HasSuffix(filePath, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	w.Write([]byte(content))
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Log the request data
	go logRequestData(r)

	requestedPath := strings.TrimPrefix(r.URL.Path, "/")
	filePath := filepath.Join(repoRoot, requestedPath)
	if requestedPath == "" {
		filePath = "web/java/Home.md"
	}

	mdContent, err := utils.GetFileContent(filePath)
	if err != nil {
		http.Error(w, "Error fetching file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	htmlContent := markdownToHTML(mdContent)

	// Get the title of the page from the first line of the Markdown file.
	lines := strings.Split(mdContent, "\n")
	title := lines[0]

	data := Data{
		Title:   title,
		Content: htmlContent,
	}

	renderTemplate(w, "layout.html", data)
}
