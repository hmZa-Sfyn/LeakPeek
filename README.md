# LeakPeek

```sh

## leakpeek.go

# build it!
go build leakpeek.go

# use it
./leakpeek "https://a.b.c" workers:10 depth:2 refmt:25-50 "link:href" files:html,js

```

## Real usage!
```sh
(myenv)  hmza@0root  ~/workspaces/hamza/LeakPeek   main  ./leakpeek https://whatever.studio workers:10 depth:2 "key:AIza[A-Za-z0-9_-]{35}" "aws:AKIA[A-Z0-9]{16}" refmt:25-25
[LeakPeek] starting → https://whatever.studio   depth:2   workers:10   context:25-25
[format] timestamp      rule    url     "match" context
---------------------------------------------------------------
2026-03-13T18:16:58Z    key     https://whatever.studio/assets/index-D9vEu-ue.js  "AIzaSyDmMtoBo8ecToDwLsKxxxxxxxxxxxxxxxx"       onst sa={GEMINI_API_KEY:"AIzaSyDmMtoBo8ecToDwLsKxxxxxxxxxxxxxxxx",DEEPSEEK_API_KEY:"sk-03

```
# What is leakpeek?
- Web recon tool to search for api leaks, and basically works like a web-grep! with builtin crawl, very fast!

# LeakPeek

**Fast, multi-threaded secret finder & recon crawler**

LeakPeek is a lightweight, high-speed web crawler designed to discover exposed secrets, API keys, tokens, credentials, private keys, JWTs, cloud credentials, payment secrets, database connection strings, and many other sensitive patterns directly in websites' HTML, JavaScript, JSON, and CSS files.

It recursively crawls a target domain (and allowed subdomains), scans content using powerful regex patterns, and displays matches **live in the terminal** as soon as they are found — no waiting for the crawl to complete.

## Core Idea

Aggressive but tunable crawling + hundreds of battle-tested secret-hunting regex patterns + real-time output with surrounding context.

Perfect balance between speed, usability, and effectiveness for modern bug bounty hunting, penetration testing, security research, and self-auditing.

## Features

- **Live findings** — matches appear instantly as they are discovered
- **Random User-Agent** rotation on every request
- **Configurable context** (`refmt:20-70` = 20 chars before + 70 after the match)
- **Depth & concurrency control** (`depth:3`, `workers:60`)
- **Body size limit** to avoid reading huge files (default ~450 KB)
- **Content-Type filtering** — only scans text/html, javascript, json, etc.
- **Simple, raw terminal output** (no fancy tables — just rule | URL | snippet)
- **Very fast** multi-threaded design (Go)
- **Hundreds of ready-to-use patterns** (AWS, GCP, Stripe, Supabase, JWT, GitHub, Twilio, SendGrid, Discord, OpenAI, and many more)

## Typical Use Cases

- **Bug bounty hunting**  
  Quickly find leaked API keys / tokens in client-side JavaScript  
  ```sh
  ./leakpeek target.com "stripe:sk_live_" "aws:AKIA" depth:3 workers:60
  ```

- **Security audits & pentests**  
  Check staging / production frontends for hardcoded credentials  
  ```sh
  ./leakpeek staging.company.com refmt:30-80
  ```

- **Post-acquisition / domain takeover checks**  
  Did the previous owner leave secrets in static assets?  
  ```sh
  ./leakpeek old-assets.example.com depth:2
  ```

- **Self-auditing your own apps/SaaS**  
  Make sure no dev accidentally committed/leaked keys to public JS/CSS  
  Run periodically or in CI/CD pipelines

- **Learning & CTF / regex practice**  
  Experiment with the included `secrets-regex.txt` list (118+ patterns)

- **Chaining with other tools**  
  Feed discovered URLs to nuclei, httpx, katana, gau, waybackurls, etc.

## Example Output

```sh
supabase_jwt  |  https://example.com/static/main.chunk.js
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c

aws_access_key  |  https://cdn.example.com/config.js
const awsKey = "AKIAJ4M3K7L9P2Q8R5T0V";

```

## Getting Started

1. Save the code as `leakpeek.go`
2. Build it:
   ```sh
   go build -o leakpeek leakpeek.go
   ```
3. Run basic scan:
   ```sh
   ./leakpeek https://target.com depth:2 workers:50 refmt:20-70 "key:AIza[A-Za-z0-9_-]{35}"
   ```
4. Use many patterns at once:
   ```sh
   ./leakpeek https://example.com depth:2 workers:60 $(grep -v '^#' secrets-regex.txt | tr '\n' ' ')
   ```

## Future Plans & Wishlist

- Save results to file / JSON / CSV
- Rate limiting & polite delays
- Colored output with severity levels
- Light headless browser mode for JS-rendered secrets
- Basic key validation (e.g., test AWS STS, Stripe ping, JWT decode)
- Proxy support & rotation
- Smarter scope filtering (`--only api.,cdn.`)
- Integration with nuclei / other scanners

## License

MIT

Happy hunting — and **always report responsibly** 🕵️‍♂️
