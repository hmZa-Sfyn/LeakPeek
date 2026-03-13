package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Rule struct {
	Name  string
	Regex *regexp.Regexp
}

var (
	client = &http.Client{Timeout: 10 * time.Second}

	linkRegex = regexp.MustCompile(`(?i)(?:href|src|data-src|data-url|action)=["']([^"'#]+)["']`)

	visited     sync.Map
	domain      string
	depthLimit  = 3
	threads     = 10
	allowedExts = map[string]bool{
		"html": true,
		"htm":  true,
		"js":   true,
		"json": true,
		"css":  true,
	}

	rules      []Rule
	ctxBefore  = 30
	ctxAfter   = 60
	results    = make(map[string][]string)
	resultsMtx sync.Mutex
)

func parseArgs() string {
	if len(os.Args) < 2 {
		fmt.Println("usage: recon-3 <url> [options]")
		fmt.Println("examples:")
		fmt.Println("  ./recon-3 https://example.com depth:3 threads:15")
		fmt.Println("  ./recon-3 https://target.com files:html,js,json \"api:AIza[A-Za-z0-9_-]{35}\"")
		os.Exit(1)
	}

	startURL := os.Args[1]

	for _, arg := range os.Args[2:] {
		switch {
		case strings.HasPrefix(arg, "depth:"):
			fmt.Sscanf(arg, "depth:%d", &depthLimit)
		case strings.HasPrefix(arg, "threads:"):
			fmt.Sscanf(arg, "threads:%d", &threads)
		case strings.HasPrefix(arg, "files:"):
			allowedExts = make(map[string]bool)
			extList := strings.TrimPrefix(arg, "files:")
			for _, e := range strings.Split(extList, ",") {
				e = strings.TrimSpace(e)
				if e != "" {
					allowedExts[e] = true
				}
			}
		case strings.HasPrefix(arg, "refmt:"):
			fmt.Sscanf(arg, "refmt:%d-%d", &ctxBefore, &ctxAfter)
		case strings.Contains(arg, ":"):
			parts := strings.SplitN(arg, ":", 2)
			if len(parts) != 2 {
				continue
			}
			name := strings.TrimSpace(parts[0])
			pattern := strings.TrimSpace(parts[1])
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Printf("Invalid regex '%s': %v\n", pattern, err)
				continue
			}
			rules = append(rules, Rule{Name: name, Regex: re})
			results[name] = []string{}
		}
	}

	return startURL
}

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")

	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		if strings.HasPrefix(raw, "//") {
			raw = "https:" + raw
		} else {
			raw = "https://" + raw
		}
	}
	return raw
}

func fetch(u string) (string, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; recon-3/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func inScope(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if strings.HasPrefix(host, "www.") {
		host = host[4:]
	}
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func extensionAllowed(u string) bool {
	if !strings.Contains(u, ".") {
		return allowedExts["html"] || allowedExts["htm"]
	}
	lower := strings.ToLower(u)
	for ext := range allowedExts {
		if strings.HasSuffix(lower, "."+ext) {
			return true
		}
	}
	return false
}

func extractLinks(base string, body string) []string {
	matches := linkRegex.FindAllStringSubmatch(body, -1)
	var links []string

	baseURL, _ := url.Parse(base)

	for _, m := range matches {
		link := m[1]
		if link == "" || strings.HasPrefix(link, "#") || strings.HasPrefix(link, "data:") {
			continue
		}

		abs, err := url.Parse(link)
		if err != nil {
			continue
		}

		abs = baseURL.ResolveReference(abs)
		absStr := abs.String()

		if inScope(absStr) && extensionAllowed(absStr) {
			links = append(links, absStr)
		}
	}
	return links
}

func getContext(body string, start, end int) string {
	s := start - ctxBefore
	e := end + ctxAfter

	if s < 0 {
		s = 0
	}
	if e > len(body) {
		e = len(body)
	}
	if s >= e {
		return ""
	}
	return body[s:e]
}

func scan(body string) {
	for _, rule := range rules {
		locs := rule.Regex.FindAllStringIndex(body, -1)
		for _, loc := range locs {
			ctx := getContext(body, loc[0], loc[1])
			if ctx == "" {
				continue
			}
			resultsMtx.Lock()
			results[rule.Name] = append(results[rule.Name], ctx)
			resultsMtx.Unlock()
		}
	}
}

func worker(queue chan struct {
	url   string
	depth int
}, wg *sync.WaitGroup) {
	for item := range queue {
		target := item.url

		if _, loaded := visited.LoadOrStore(target, struct{}{}); loaded {
			wg.Done()
			continue
		}

		body, err := fetch(target)
		if err != nil {
			wg.Done()
			continue
		}

		scan(body)

		if item.depth < depthLimit {
			links := extractLinks(target, body)
			for _, link := range links {
				if _, loaded := visited.Load(link); !loaded {
					wg.Add(1)
					queue <- struct {
						url   string
						depth int
					}{link, item.depth + 1}
				}
			}
		}

		wg.Done()
	}
}

func printResults() {
	if len(results) == 0 {
		fmt.Println("No rules were defined or no matches found.")
		return
	}

	foundAny := false
	for name, matches := range results {
		if len(matches) == 0 {
			continue
		}
		foundAny = true

		maxLen := len(name)
		for _, m := range matches {
			if l := len(m); l > maxLen {
				maxLen = l
			}
		}
		if maxLen < 60 {
			maxLen = 60
		}

		sep := strings.Repeat("-", maxLen+4)
		fmt.Printf("+%s+\n", sep)
		fmt.Printf("| %-*s |\n", maxLen, name)
		fmt.Printf("+%s+\n", sep)

		for _, match := range matches {
			display := match
			if len(display) > maxLen+30 {
				display = display[:maxLen+27] + "..."
			}
			fmt.Printf("| %-*s |\n", maxLen, display)
		}

		fmt.Printf("+%s+\n\n", sep)
	}

	if !foundAny {
		fmt.Println("No matches found for any of the defined rules.")
	}
}

func main() {
	start := parseArgs()
	start = normalizeURL(start)

	// Validate URL and set domain
	parsedURL, err := url.Parse(start)
	if err != nil {
		fmt.Println("Invalid starting URL:", err)
		os.Exit(1)
	}

	domain = parsedURL.Hostname()
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	if len(rules) == 0 {
		fmt.Println("Warning: No search rules provided (example: secret:AIza[A-Za-z0-9_-]{35})")
	}

	fmt.Printf("┌ Starting crawl\n")
	fmt.Printf("├─ Target : %s\n", start)
	fmt.Printf("├─ Domain : %s\n", domain)
	fmt.Printf("├─ Depth  : %d\n", depthLimit)
	fmt.Printf("├─ Threads: %d\n", threads)
	fmt.Printf("└─ Files  : %v\n", allowedExts)

	queue := make(chan struct {
		url   string
		depth int
	}, 20000)

	var wg sync.WaitGroup

	for i := 0; i < threads; i++ {
		go worker(queue, &wg)
	}

	wg.Add(1)
	queue <- struct {
		url   string
		depth int
	}{start, 0}

	wg.Wait()
	close(queue)

	printResults()
}