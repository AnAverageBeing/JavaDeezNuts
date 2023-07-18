package main

import (
	"JavaDeezNuts/utils"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

type AccessLog struct {
	Time        string `json:"time"`
	IP          string `json:"ip"`
	Path        string `json:"path"`
	UserAgent   string `json:"user_agent"`
	GeoLocation string `json:"geo_location"`
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

func logRequestData(r *http.Request) {
	requestTime := time.Now().Format(time.ANSIC)

	// Get the user's IP address from the CF-Connecting-IP header
	ipAddress := r.Header.Get("CF-Connecting-IP")
	if ipAddress == "" {
		// If CF-Connecting-IP header is not available, fallback to RemoteAddr
		ipAddress, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	userAgent := r.UserAgent()

	requestedPath := strings.TrimPrefix(r.URL.Path, "/")

	geoLocation := getGeoLocation(ipAddress)

	logData := AccessLog{
		Time:        requestTime,
		IP:          ipAddress,
		Path:        requestedPath,
		UserAgent:   userAgent,
		GeoLocation: geoLocation,
	}

	fmt.Printf("[%s] IP: %s Requested: %s, GeoLocation: %s\n",
		requestTime, ipAddress, requestedPath, geoLocation)

	writeAccessLog(logData)
}

func getGeoLocation(ipAddress string) string {
	response, err := http.Get(ipAPIEndpoint + ipAddress)
	if err != nil {
		log.Printf("Error fetching geo-location: %s", err)
		return "Unknown"
	}
	defer response.Body.Close()

	var geoInfo struct {
		Success string `json:"status"`
		City    string `json:"city"`
		Region  string `json:"regionName"`
		Country string `json:"country"`
	}

	err = json.NewDecoder(response.Body).Decode(&geoInfo)
	if err != nil {
		log.Printf("Error parsing geo-location data: %s", err)
		return "Unknown"
	}

	if strings.Compare(geoInfo.Success, "success") != 0 {
		return "UNABLE TO GET"
	}

	geoLocation := fmt.Sprintf("%s, %s, %s", geoInfo.City, geoInfo.Region, geoInfo.Country)
	return geoLocation
}

func writeAccessLog(data AccessLog) {
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Error opening log file: %s", err)
		return
	}
	defer logFile.Close()

	logData, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Printf("Error marshaling access log data: %s", err)
		return
	}

	logFile.Write(logData)
	logFile.Write([]byte("\n"))
}

func main() {
	port := flag.Int("p", 8080, "Port number to run the server on")
	useHTTP2 := flag.Bool("http2", true, "Enable HTTP/2")
	flag.Parse()

	http.HandleFunc("/", handler)
	http.HandleFunc("/static/", handleStaticFile)

	utils.StartWebhookReceiver()

	if *useHTTP2 {
		http2Enabled := &http.Server{
			Addr:    fmt.Sprintf(":%d", *port),
			Handler: nil, // The default handler will be used.
			TLSConfig: &tls.Config{
				NextProtos: []string{"h2", "http/1.1"},
			},
		}
		log.Printf("Starting server with HTTP/2 on http://localhost:%d\n", *port)
		log.Fatal(http2Enabled.ListenAndServe())
		return
	}

	fmt.Printf("Starting server on http://localhost:%d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
