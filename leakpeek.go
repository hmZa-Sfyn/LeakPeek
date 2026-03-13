package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/128.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Edg/129.0.0.0 Safari/537.36",
	}

	rnd        = rand.New(rand.NewSource(time.Now().UnixNano()))
	rndMu      sync.Mutex

	client     = &http.Client{Timeout: 5 * time.Second}
	linkRegex  = regexp.MustCompile(`(?i)(?:href|src|data-(?:src|url))=["']([^"'#]+)["']`)

	visited     sync.Map
	domain      string
	maxDepth    = 3
	workers     = 50
	maxBody     = 450 * 1024       // 450 KB cap

	ctxBefore   = 30               // default
	ctxAfter    = 65               // default

	totalVisited atomic.Int32
	matchesFound atomic.Int32
	printMu      sync.Mutex

	allowedExts = map[string]bool{"html": true, "htm": true, "js": true, "json": true}
)

type Rule struct {
	Name  string
	Regex *regexp.Regexp
}

var globalRules []Rule

func randUA() string {
	rndMu.Lock()
	defer rndMu.Unlock()
	return userAgents[rnd.Intn(len(userAgents))]
}

func normalizeURL(s string) string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	return strings.TrimRight(s, "/")
}

func parseArgs() string {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ./recon <url> [options] [rule:pattern ...]")
		fmt.Fprintln(os.Stderr, "options:")
		fmt.Fprintln(os.Stderr, "  depth:N           max crawl depth")
		fmt.Fprintln(os.Stderr, "  workers:N         number of concurrent workers")
		fmt.Fprintln(os.Stderr, "  refmt:XX-YY       chars before-after match (context)")
		os.Exit(1)
	}

	start := normalizeURL(os.Args[1])

	for _, arg := range os.Args[2:] {
		switch {
		case strings.HasPrefix(arg, "depth:"):
			fmt.Sscanf(arg, "depth:%d", &maxDepth)
		case strings.HasPrefix(arg, "workers:"):
			fmt.Sscanf(arg, "workers:%d", &workers)
		case strings.HasPrefix(arg, "refmt:"):
			var b, a int
			_, err := fmt.Sscanf(arg, "refmt:%d-%d", &b, &a)
			if err == nil && b >= 0 && a >= 0 {
				ctxBefore = b
				ctxAfter = a
			}
		default:
			parts := strings.SplitN(arg, ":", 2)
			if len(parts) != 2 {
				continue
			}
			name := strings.TrimSpace(parts[0])
			pat := strings.TrimSpace(parts[1])
			re, err := regexp.Compile(pat)
			if err != nil {
				fmt.Fprintf(os.Stderr, "bad regex %q → %v\n", pat, err)
				continue
			}
			globalRules = append(globalRules, Rule{name, re})
		}
	}

	return start
}

func fetch(u string) (string, error) {
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", randUA())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(ct, "html") &&
		!strings.Contains(ct, "javascript") &&
		!strings.Contains(ct, "json") {
		return "", nil // skip non-text
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBody)))
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", nil
	}

	return string(body), nil
}

func inScopeFast(s string) bool {
	return strings.Contains(s, domain)
}

func extractLinks(base, body string) []string {
	ms := linkRegex.FindAllStringSubmatch(body, -1)
	var links []string
	baseURL, _ := url.Parse(base)

	for _, m := range ms {
		if len(m) < 2 {
			continue
		}
		link := m[1]
		if link == "" || link[0] == '#' || strings.HasPrefix(link, "data:") {
			continue
		}
		abs, err := url.Parse(link)
		if err != nil {
			continue
		}
		abs = baseURL.ResolveReference(abs)
		s := abs.String()
		if inScopeFast(s) {
			links = append(links, s)
		}
	}
	return links
}

func showMatch(ruleName, pageURL, context string) {
	printMu.Lock()
	defer printMu.Unlock()

	fmt.Printf("%s  |  %s\n", ruleName, pageURL)
	fmt.Println(context)
	fmt.Println("---")
	matchesFound.Add(1)
}

func scan(body, pageURL string) {
	for _, r := range globalRules {
		locs := r.Regex.FindAllStringIndex(body, -1)
		for _, loc := range locs {
			s := max(0, loc[0]-ctxBefore)
			e := min(len(body), loc[1]+ctxAfter)
			if s >= e {
				continue
			}
			showMatch(r.Name, pageURL, body[s:e])
		}
	}
}

func worker(q chan struct{ url string; depth int }, wg *sync.WaitGroup) {
	for item := range q {
		u := item.url

		if _, seen := visited.LoadOrStore(u, true); seen {
			wg.Done()
			continue
		}

		totalVisited.Add(1)

		body, err := fetch(u)
		if err != nil || body == "" {
			wg.Done()
			continue
		}

		scan(body, u)

		if item.depth < maxDepth {
			next := extractLinks(u, body)
			for _, ln := range next {
				if _, seen := visited.Load(ln); !seen {
					wg.Add(1)
					q <- struct{ url string; depth int }{ln, item.depth + 1}
				}
			}
		}

		wg.Done()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	start := parseArgs()

	u, err := url.Parse(start)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid url")
		os.Exit(1)
	}
	domain = u.Hostname()
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	if len(globalRules) == 0 {
		fmt.Fprintln(os.Stderr, "warning: no patterns given")
	}

	fmt.Printf("→ %s   depth:%d   workers:%d   context:%d-%d\n\n",
		start, maxDepth, workers, ctxBefore, ctxAfter)

	queue := make(chan struct{ url string; depth int }, 40000)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(queue, &wg)
	}

	wg.Add(1)
	queue <- struct{ url string; depth int }{start, 0}

	go func() {
		wg.Wait()
		close(queue)
	}()

	wg.Wait()

	fmt.Printf("\nFinished → visited: %d   matches: %d\n",
		totalVisited.Load(), matchesFound.Load())
}