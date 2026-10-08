package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests share application globals and must not run in parallel.
func resetLinks(t *testing.T) {
	t.Helper()
	clear := func() {
		links.Range(func(key, value interface{}) bool {
			links.Delete(key)
			return true
		})
	}
	clear()
	t.Cleanup(clear)
}

func TestAddLink(t *testing.T) {
	resetLinks(t)
	queue := make(chan string, 2)
	link := "https://example.com/path?q=one&other=two"
	ok, id := addLink(link, queue)
	if !ok {
		t.Fatalf("addLink failed: %s", id)
	}
	if got := getLink(id); got != link {
		t.Fatalf("stored URL = %q, want %q", got, link)
	}
	select {
	case got := <-queue:
		if want := fmt.Sprintf(format, id, link); got != want {
			t.Fatalf("queued record = %q, want %q", got, want)
		}
	default:
		t.Fatal("new link was not queued for saving")
	}
	if ok, got := addLink(link, queue); !ok || got != id {
		t.Fatalf("duplicate URL returned (%v, %q), want (true, %q)", ok, got, id)
	}
	if len(queue) != 0 {
		t.Fatal("duplicate URL was queued again")
	}
}

func TestAddLinkRejectsInvalidURL(t *testing.T) {
	for _, link := range []string{"http://%zz", "http://[::1", "https://example.com/\n"} {
		t.Run(link, func(t *testing.T) {
			resetLinks(t)
			queue := make(chan string, 1)
			ok, message := addLink(link, queue)
			if ok || !strings.Contains(message, "Error parsing link:") {
				t.Fatalf("invalid URL returned (%v, %q)", ok, message)
			}
			if len(queue) != 0 || getLink(getHash(link)) != "" {
				t.Fatal("invalid URL was stored or queued")
			}
		})
	}
}

func TestAddLinkRejectsCollision(t *testing.T) {
	resetLinks(t)
	first := "https://example.com/69897/4885590609"
	second := "https://example.com/105844/11202952336"
	queue := make(chan string, 2)
	ok, id := addLink(first, queue)
	if !ok {
		t.Fatalf("first URL rejected: %s", id)
	}
	if ok, message := addLink(second, queue); ok || message == "" {
		t.Fatalf("collision returned (%v, %q), want rejection", ok, message)
	}
	if getLink(id) != first || len(queue) != 1 {
		t.Fatal("collision replaced the original URL or queued another record")
	}
}

func TestAddLinkConcurrentDuplicates(t *testing.T) {
	resetLinks(t)
	const workers = 32
	queue := make(chan string, workers)
	link := "https://example.com/concurrent"
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, id := addLink(link, queue); !ok || id != getHash(link) {
				t.Errorf("addLink returned (%v, %q)", ok, id)
			}
		}()
	}
	wg.Wait()
	if len(queue) != 1 || getLink(getHash(link)) != link {
		t.Fatal("concurrent duplicates should store and queue exactly one mapping")
	}
}

func TestGetLinkMissing(t *testing.T) {
	resetLinks(t)
	if got := getLink("missing"); got != "" {
		t.Fatalf("missing URL = %q, want empty string", got)
	}
}

func TestReadLinks(t *testing.T) {
	resetLinks(t)
	path := filepath.Join(t.TempDir(), "links.txt")
	content := "one<>https://example.com/first\ntwo<>https://example.com/?q=<>\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	readLinks(path)
	for id, want := range map[string]string{
		"one": "https://example.com/first",
		"two": "https://example.com/?q=<>",
	} {
		if got := getLink(id); got != want {
			t.Fatalf("getLink(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestReadLinksCreatesMissingFile(t *testing.T) {
	resetLinks(t)
	path := filepath.Join(t.TempDir(), "links.txt")
	readLinks(path)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatal("new storage file should be empty")
	}
}

func TestSaveLinkAppendsAndReloads(t *testing.T) {
	resetLinks(t)
	path := filepath.Join(t.TempDir(), "links.txt")
	initial := "old<>https://example.com/old\n"
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}
	queue := make(chan string, 2)
	queue <- "new<>https://example.com/new\n"
	queue <- "query<>https://example.com/?q=<>\n"
	close(queue)
	done := make(chan struct{})
	go func() {
		saveLink(path, queue)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("saveLink did not stop after the queue closed")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := initial + "new<>https://example.com/new\nquery<>https://example.com/?q=<>\n"
	if string(content) != want {
		t.Fatalf("saved content = %q, want %q", content, want)
	}
	readLinks(path)
	for id, want := range map[string]string{
		"old":   "https://example.com/old",
		"new":   "https://example.com/new",
		"query": "https://example.com/?q=<>",
	} {
		if got := getLink(id); got != want {
			t.Fatalf("reloaded %q = %q, want %q", id, got, want)
		}
	}
}
