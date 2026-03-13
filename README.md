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
(myenv)  ✘ hmza@0root  ~/workspaces/hamza/LeakPeek   main  ./leakpeek https://localhost:5000 workers:10 depth:2 "key:AIza[A-Za-z0-9_-]{35}" "aws:AKIA[A-Z0-9]{16}" refmt:25-25 
→ https://localhost:5000   depth:2   workers:10   context:25-25

key  |  https://localhost:5000/assets/index-D9vEu-ue.js
onst sa={GEMINI_API_KEY:"AIzaSyDmMtoBo8ecToDwLsK6xxxxxxxxxxxxxxx",DEEPSEEK_API_KEY:"sk-03
---
^C
(myenv)  ✘ hmza@0root  ~/workspaces/hamza/LeakPeek   main  
```

Web recon tool to search for api leaks, and basically works like a web-grep! with builtin crawl, very fast!
