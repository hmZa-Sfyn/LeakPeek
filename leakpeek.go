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
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Edg/129.0.0.0",
	}

	rnd        = rand.New(rand.NewSource(time.Now().UnixNano()))
	rndMu      sync.Mutex

	client     = &http.Client{Timeout: 5 * time.Second}
	linkRegex  = regexp.MustCompile(`(?i)(?:href|src|data-(?:src|url|original))=["']([^"'#]+)["']`)

	visited     sync.Map
	domain      string
	maxDepth    = 3
	workers     = 50
	maxBody     = 450 * 1024       // 450 KB limit

	ctxBefore   = 30
	ctxAfter    = 65

	totalVisited atomic.Int32
	matchesFound atomic.Int32
	printMu      sync.Mutex

	allowedExts = map[string]bool{
		"html": true, "htm": true,
		"js":   true, "json": true,
		"css":  true,
	}
)

type Rule struct {
	Name  string
	Regex *regexp.Regexp
}

var rules []Rule

func randUserAgent() string {
	rndMu.Lock()
	defer rndMu.Unlock()
	return userAgents[rnd.Intn(len(userAgents))]
}

func normalizeURL(input string) string {
	input = strings.TrimSpace(input)
	input = strings.TrimRight(input, "/")
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		if strings.HasPrefix(input, "//") {
			input = "https:" + input
		} else {
			input = "https://" + input
		}
	}
	return input
}

func parseArgs() string {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "LeakPeek - fast secret finder & recon crawler")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  ./leakpeek <url> [options] [rule:pattern ...]")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  depth:N           max crawl depth (default 3)")
		fmt.Fprintln(os.Stderr, "  workers:N         concurrent workers (default 50)")
		fmt.Fprintln(os.Stderr, "  refmt:XX-YY       chars before-after match (default 30-65)")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  ./leakpeek example.com \"aws:AKIA[A-Z0-9]{16}\" depth:2")
		fmt.Fprintln(os.Stderr, "  ./leakpeek target.com refmt:20-80 workers:70")
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
			pattern := strings.TrimSpace(parts[1])
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid regex for %q: %v\n", name, err)
				continue
			}
			rules = append(rules, Rule{Name: name, Regex: re})
		}
	}

	return start
}

func fetch(target string) (string, error) {
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", randUserAgent())
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
		return "", nil // skip non-interesting content
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

func inScope(link string) bool {
	return strings.Contains(link, domain)
}

func extractLinks(baseURL, body string) []string {
	matches := linkRegex.FindAllStringSubmatch(body, -1)
	var links []string

	base, _ := url.Parse(baseURL)

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := m[1]
		if raw == "" || raw[0] == '#' || strings.HasPrefix(raw, "data:") {
			continue
		}
		abs, err := url.Parse(raw)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(abs)
		s := resolved.String()
		if inScope(s) {
			links = append(links, s)
		}
	}
	return links
}

func reportFinding(ruleName, pageURL, exactMatch, context string) {
	printMu.Lock()
	defer printMu.Unlock()

	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	cleanCtx := strings.ReplaceAll(strings.TrimSpace(context), "\n", " ")
	cleanMatch := strings.TrimSpace(exactMatch)

	// Tab-separated: timestamp  rule  url  "match"  context
	fmt.Printf("%s\t%s\t%s\t%q\t%s\n",
		ts,
		ruleName,
		pageURL,
		cleanMatch,
		cleanCtx,
	)

	matchesFound.Add(1)
}

func scan(content, pageURL string) {
	for _, rule := range rules {
		locs := rule.Regex.FindAllStringIndex(content, -1)
		for _, loc := range locs {
			s := max(0, loc[0]-ctxBefore)
			e := min(len(content), loc[1]+ctxAfter)
			if s >= e {
				continue
			}
			ctx := content[s:e]
			match := content[loc[0]:loc[1]]
			reportFinding(rule.Name, pageURL, match, ctx)
		}
	}
}

func worker(queue chan struct{ url string; depth int }, wg *sync.WaitGroup) {
	for item := range queue {
		target := item.url

		if _, seen := visited.LoadOrStore(target, true); seen {
			wg.Done()
			continue
		}

		totalVisited.Add(1)

		body, err := fetch(target)
		if err != nil || body == "" {
			wg.Done()
			continue
		}

		scan(body, target)

		if item.depth < maxDepth {
			nextLinks := extractLinks(target, body)
			for _, link := range nextLinks {
				if _, seen := visited.Load(link); !seen {
					wg.Add(1)
					queue <- struct{ url string; depth int }{link, item.depth + 1}
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
	startURL := parseArgs()

	u, err := url.Parse(startURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid URL:", err)
		os.Exit(1)
	}

	domain = u.Hostname()
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}

	if len(rules) == 0 {
		fmt.Fprintln(os.Stderr, "Warning: no search rules provided")
	}

	fmt.Fprintf(os.Stderr, "[LeakPeek] starting → %s   depth:%d   workers:%d   context:%d-%d\n",
		startURL, maxDepth, workers, ctxBefore, ctxAfter)
	fmt.Fprintln(os.Stderr, "[format] timestamp\trule\turl\t\"match\"\tcontext")
	fmt.Fprintln(os.Stderr, "---------------------------------------------------------------")

	queue := make(chan struct{ url string; depth int }, 40000)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(queue, &wg)
	}

	wg.Add(1)
	queue <- struct{ url string; depth int }{startURL, 0}

	go func() {
		wg.Wait()
		close(queue)
	}()

	wg.Wait()

	fmt.Fprintf(os.Stderr, "\nFinished. Visited: %d   Matches found: %d\n",
		totalVisited.Load(), matchesFound.Load())
}