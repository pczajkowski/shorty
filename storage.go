package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
)

const (
	format = "%s<>%s\n"
)

var links sync.Map

func readLinks(path string) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		log.Fatalf("Failed to open %s!\n", path)

	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("Failed to close file: %s", err)
		}
	}()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}

		parts := strings.Split(line, "<>")
		if len(parts) != 2 {
			log.Printf("Wrong line format: %s", line)
		}

		links.Store(parts[0], parts[1])
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %s", err)
	}

}

func addLink(link string, toSave chan<- string) string {
	u, err := url.Parse(link)
	if err != nil {
		log.Printf("Error parsing link: %s", err)
	}
	link = u.String()

	linkID := getHash(link)

	existingLink, loaded := links.LoadOrStore(linkID, link)
	if loaded {
		if existingLink != link {
			log.Printf("Have collision:\n%s\n%s\n", link, existingLink)
		}

		return linkID
	}

	toSave <- fmt.Sprintf(format, linkID, link)
	return linkID
}

func getLink(linkID string) string {
	link, found := links.Load(linkID)
	if !found {
		return ""
	}

	return link.(string)
}

func saveLink(path string, toSave <-chan string) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("Failed to close file: %s", err)
		}
	}()

	for item := range toSave {
		if _, err := file.WriteString(item); err != nil {
			log.Println(err)
		}
	}
}
