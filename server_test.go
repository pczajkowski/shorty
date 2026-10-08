package main

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func resetServer(t *testing.T) {
	t.Helper()
	resetLinks(t)
	previousQueue, previousDomain := toSave, *domain
	toSave = make(chan string, 10)
	*domain = ""
	t.Cleanup(func() {
		toSave, *domain = previousQueue, previousDomain
	})
}

func TestShorten(t *testing.T) {
	link := "https://example.com/path"
	for _, tc := range []struct{ name, target, domain, prefix string }{
		{"query", "/s/?link=" + url.QueryEscape(link), "", "short.test/"},
		{"path", "/s/" + link, "", "short.test/"},
		{"configured domain", "/s/?link=" + url.QueryEscape(link), "https://short.test", "https://short.test/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetServer(t)
			*domain = tc.domain
			r := httptest.NewRequest(http.MethodGet, tc.target, nil)
			r.Host = "short.test"
			w := httptest.NewRecorder()
			shorten(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
			want := `value="` + tc.prefix + getHash(link) + `"`
			if !strings.Contains(w.Body.String(), want) || getLink(getHash(link)) != link {
				t.Fatalf("shortened link missing or not stored: %s", w.Body.String())
			}
			if len(toSave) != 1 {
				t.Fatal("shorten did not queue exactly one record")
			}
		})
	}
}

func TestShortenRejectsInvalidURL(t *testing.T) {
	resetServer(t)
	w := httptest.NewRecorder()
	shorten(w, httptest.NewRequest(http.MethodGet, "/s/?link="+url.QueryEscape("http://%zz"), nil))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Error parsing link:") {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if len(toSave) != 0 {
		t.Fatal("invalid URL was queued")
	}
}

func TestDecode(t *testing.T) {
	for _, target := range []string{"/d/abc", "/d/?link=abc", "/d/?link=" + url.QueryEscape("https://short.test/abc")} {
		t.Run(target, func(t *testing.T) {
			resetServer(t)
			link := "https://example.com/?a=1&b=2"
			links.Store("abc", link)
			w := httptest.NewRecorder()
			decode(w, httptest.NewRequest(http.MethodGet, target, nil))
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `value="`+html.EscapeString(link)+`"`) {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestDecodeMissing(t *testing.T) {
	for _, target := range []string{"/d/", "/d/missing", "/d/?link=https://short.test/missing"} {
		t.Run(target, func(t *testing.T) {
			resetServer(t)
			w := httptest.NewRecorder()
			decode(w, httptest.NewRequest(http.MethodGet, target, nil))
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Not found!") {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestRedirectOrServe(t *testing.T) {
	t.Run("known ID redirects", func(t *testing.T) {
		resetServer(t)
		link := "https://example.com/path?q=one&other=two"
		links.Store("abc", link)
		w := httptest.NewRecorder()
		redirectOrServe(w, httptest.NewRequest(http.MethodGet, "/abc", nil))
		if w.Code != http.StatusMovedPermanently || w.Header().Get("Location") != link {
			t.Fatalf("status = %d, Location = %q", w.Code, w.Header().Get("Location"))
		}
	})
	for _, target := range []string{"/", "/missing"} {
		t.Run(target, func(t *testing.T) {
			resetServer(t)
			w := httptest.NewRecorder()
			redirectOrServe(w, httptest.NewRequest(http.MethodGet, target, nil))
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Link to shorten:") {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestReturnErrorEscapesHTML(t *testing.T) {
	message := `<script>alert("oops")</script>`
	w := httptest.NewRecorder()
	returnErorr(w, message)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), html.EscapeString(message)) {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), message) {
		t.Fatal("error message was rendered as executable HTML")
	}
}
