package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var visited sync.Map

var endpointRegex = regexp.MustCompile(`https?://[^\s"'<>]+|\/[a-zA-Z0-9_\-\/]+`)
var paramRegex = regexp.MustCompile(`\?([a-zA-Z0-9_\-]+)=`)
var secretRegex = regexp.MustCompile(`(?i)(apikey|token|secret|password|auth)[\s"':=]+[a-zA-Z0-9_\-]{8,}`)

var linkRegex = regexp.MustCompile(`(href|src)=["']([^"'#]+)["']`)

var scopeDomain string

func inScope(raw string) bool {

	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	host := u.Hostname()

	return host == scopeDomain || strings.HasSuffix(host, "."+scopeDomain)
}

func fetch(u string) (string, error) {

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "fast-recon-bot")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	var builder strings.Builder

	for scanner.Scan() {
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func extractLinks(base string, body string) []string {

	matches := linkRegex.FindAllStringSubmatch(body, -1)

	var links []string

	for _, m := range matches {

		link := m[2]

		u, err := url.Parse(link)
		if err != nil {
			continue
		}

		baseURL, _ := url.Parse(base)

		abs := baseURL.ResolveReference(u)

		if inScope(abs.String()) {
			links = append(links, abs.String())
		}
	}

	return links
}

func analyzeContent(target string, body string) {

	endpoints := endpointRegex.FindAllString(body, -1)

	for _, e := range endpoints {
		fmt.Println("[endpoint]", target, "->", e)
	}

	params := paramRegex.FindAllStringSubmatch(body, -1)

	for _, p := range params {
		fmt.Println("[param]", target, "->", p[1])
	}

	secrets := secretRegex.FindAllString(body, -1)

	for _, s := range secrets {
		fmt.Println("[secret]", target, "->", s)
	}
}

func worker(queue chan string, wg *sync.WaitGroup) {

	for u := range queue {

		if _, ok := visited.LoadOrStore(u, true); ok {
			wg.Done()
			continue
		}

		body, err := fetch(u)

		if err == nil {

			fmt.Println("[page]", u)

			analyzeContent(u, body)

			links := extractLinks(u, body)

			for _, l := range links {

				if _, ok := visited.Load(l); !ok {
					wg.Add(1)
					queue <- l
				}
			}
		}

		wg.Done()
	}
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("usage: recon <url>")
		return
	}

	start := os.Args[1]

	u, err := url.Parse(start)
	if err != nil {
		fmt.Println("invalid url")
		return
	}

	scopeDomain = u.Hostname()

	queue := make(chan string, 10000)

	var wg sync.WaitGroup

	workers := 50

	for i := 0; i < workers; i++ {
		go worker(queue, &wg)
	}

	wg.Add(1)
	queue <- start

	wg.Wait()

	close(queue)
}