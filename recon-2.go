package main

import (
	"flag"
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

var (
	startURL string

	flagSecrets   bool
	flagEndpoints bool
	flagParams    bool
	flagStrings   bool
	flagForms     bool
	flagOptions   bool
	flagAll       bool

	scopeDomain string
	client      = &http.Client{Timeout: 10 * time.Second}

	workers = 40

	visited sync.Map
)

var linkRegex = regexp.MustCompile(`(href|src)=["']([^"'#]+)["']`)

var endpointRegex = regexp.MustCompile(`https?://[^\s"'<>]+|\/[a-zA-Z0-9_\-\/]+`)

var paramRegex = regexp.MustCompile(`\?([a-zA-Z0-9_\-]+)=`)

var stringRegex = regexp.MustCompile(`"([^"]{4,})"|'([^']{4,})'`)

var secretRegex = regexp.MustCompile(`(?i)(
AKIA[0-9A-Z]{16}|
AIza[0-9A-Za-z\-_]{35}|
sk_live_[0-9a-zA-Z]{24}|
api[_-]?key[\s"':=]+[0-9A-Za-z_\-]{8,}|
token[\s"':=]+[0-9A-Za-z_\-]{8,}|
secret[\s"':=]+[0-9A-Za-z_\-]{8,}|
password[\s"':=]+[0-9A-Za-z_\-]{6,}
)`)

func initFlags() {

	flag.StringVar(&startURL, "url", "", "target url")

	flag.BoolVar(&flagSecrets, "secrets", false, "scan secrets")
	flag.BoolVar(&flagEndpoints, "endpoints", false, "scan endpoints")
	flag.BoolVar(&flagParams, "params", false, "scan parameters")
	flag.BoolVar(&flagStrings, "strings", false, "extract strings")
	flag.BoolVar(&flagForms, "forms", false, "scan forms")
	flag.BoolVar(&flagOptions, "options", false, "send OPTIONS requests")
	flag.BoolVar(&flagAll, "all", false, "enable all scans")

	flag.Parse()

	if flagAll {
		flagSecrets = true
		flagEndpoints = true
		flagParams = true
		flagStrings = true
		flagForms = true
		flagOptions = true
	}

	if startURL == "" {
		fmt.Println("usage: recon -url https://target.com [flags]")
		os.Exit(1)
	}
}

func write(file, line string) {

	os.MkdirAll("scan_output", 0755)

	f, _ := os.OpenFile("scan_output/"+file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	f.WriteString(line + "\n")
}

func inScope(raw string) bool {

	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	host := u.Hostname()

	return host == scopeDomain || strings.HasSuffix(host, "."+scopeDomain)
}

func fetch(target string) (string, error) {

	req, _ := http.NewRequest("GET", target, nil)
	req.Header.Set("User-Agent", "webrecon")

	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(body), nil
}

func scanEndpoints(target, body string) {

	if !flagEndpoints {
		return
	}

	matches := endpointRegex.FindAllString(body, -1)

	for _, e := range matches {
		write("endpoints.txt", target+" -> "+e)
	}
}

func scanParams(target, body string) {

	if !flagParams {
		return
	}

	params := paramRegex.FindAllStringSubmatch(body, -1)

	for _, p := range params {
		write("parameters.txt", target+" -> "+p[1])
	}
}

func scanStrings(target, body string) {

	if !flagStrings {
		return
	}

	matches := stringRegex.FindAllStringSubmatch(body, -1)

	for _, m := range matches {

		if m[1] != "" {
			write("strings.txt", target+" -> "+m[1])
		}

		if m[2] != "" {
			write("strings.txt", target+" -> "+m[2])
		}
	}
}

func scanSecrets(target, body string) {

	if !flagSecrets {
		return
	}

	matches := secretRegex.FindAllString(body, -1)

	for _, m := range matches {
		write("secrets.txt", target+" -> "+m)
	}
}

func scanForms(target, body string) {

	if !flagForms {
		return
	}

	formRegex := regexp.MustCompile(`(?is)<form[^>]*>(.*?)</form>`)
	actionRegex := regexp.MustCompile(`action=["']([^"']+)`)
	methodRegex := regexp.MustCompile(`method=["']([^"']+)`)
	inputRegex := regexp.MustCompile(`name=["']([^"']+)`)

	forms := formRegex.FindAllString(body, -1)

	for _, f := range forms {

		action := actionRegex.FindStringSubmatch(f)
		method := methodRegex.FindStringSubmatch(f)
		inputs := inputRegex.FindAllStringSubmatch(f, -1)

		act := target
		meth := "GET"

		if len(action) > 1 {
			act = action[1]
		}

		if len(method) > 1 {
			meth = strings.ToUpper(method[1])
		}

		write("forms.txt", target+" -> "+act+" ["+meth+"]")

		var params []string

		for _, in := range inputs {
			params = append(params, in[1]+"=test")
		}

		query := strings.Join(params, "&")

		if meth == "GET" {
			write("form_requests.txt", "GET "+act+"?"+query)
		} else {
			write("form_requests.txt", "POST "+act+" BODY:"+query)
		}
	}
}

func scanOptions(target string) {

	if !flagOptions {
		return
	}

	req, _ := http.NewRequest("OPTIONS", target, nil)

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	defer resp.Body.Close()

	allow := resp.Header.Get("Allow")

	if allow != "" {
		write("options.txt", target+" -> "+allow)
	}
}

func extractLinks(base, body string) []string {

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

func worker(queue chan string, wg *sync.WaitGroup) {

	for target := range queue {

		if _, ok := visited.LoadOrStore(target, true); ok {
			wg.Done()
			continue
		}

		write("pages.txt", target)

		body, err := fetch(target)

		if err == nil {

			scanEndpoints(target, body)
			scanParams(target, body)
			scanStrings(target, body)
			scanSecrets(target, body)
			scanForms(target, body)

			scanOptions(target)

			links := extractLinks(target, body)

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

	initFlags()

	u, err := url.Parse(startURL)
	if err != nil {
		fmt.Println("invalid url")
		return
	}

	scopeDomain = u.Hostname()

	queue := make(chan string, 10000)

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		go worker(queue, &wg)
	}

	wg.Add(1)
	queue <- startURL

	wg.Wait()

	close(queue)
}