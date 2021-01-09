package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"strings"
)

const (
	shortenPath = "/s/"
	decodePath  = "/d/"
	indexHTML   = "./html/index.html"
)

var host = flag.String("h", "localhost", "Host on which to serve")
var port = flag.String("p", "9090", "Port on which to serve")
var file = flag.String("f", "links.txt", "File to which save links")
var domain = flag.String("d", "", "Domain of shorty, preferably add schema")

var toSave chan string

func init() {
	toSave = make(chan string, 100)
}

func shorten(w http.ResponseWriter, r *http.Request) {
	link := r.URL.Query().Get("link")
	if link == "" {
		link = strings.TrimPrefix(r.URL.Path, shortenPath)
	}

	linkID := addLink(link, toSave)

	t := template.Must(template.ParseFiles("./html/result.html"))

	shortened := r.Host + "/" + linkID
	if *domain != "" {
		shortened = *domain + "/" + linkID
	}
	t.Execute(w, shortened)
}

func decode(w http.ResponseWriter, r *http.Request) {
	link := r.URL.Query().Get("link")
	if link == "" {
		link = strings.TrimPrefix(r.URL.Path, shortenPath)
	}

	t := template.Must(template.ParseFiles("./html/result.html"))

	parts := strings.Split(link, "/")
	linkID := parts[len(parts)-1]
	if linkID != "" {
		fullLink := getLink(linkID)
		if fullLink != "" {
			t.Execute(w, fullLink)
			return
		}
	}

	t.Execute(w, "Not found!")
}

func redirectOrServe(w http.ResponseWriter, r *http.Request) {
	linkID := strings.TrimPrefix(r.URL.Path, "/")

	if linkID == "" {
		http.ServeFile(w, r, indexHTML)
	} else {
		link := getLink(linkID)
		if link != "" {
			http.Redirect(w, r, link, http.StatusMovedPermanently)
		} else {
			http.ServeFile(w, r, indexHTML)
		}
	}
}

func main() {
	flag.Parse()
	readLinks(*file)

	go saveLink(*file, toSave)

	hostname := *host + ":" + *port
	http.HandleFunc(shortenPath, shorten)
	http.HandleFunc(decodePath, decode)
	http.HandleFunc("/", redirectOrServe)
	log.Fatal(http.ListenAndServe(hostname, nil))
}
