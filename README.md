# LeakPeek

```sh

## leakpeek.go

# build it!
go build leakpeek.go

# use it
./leakpeek https://donjon.studio workers:10 depth:2 refmt:25-50 "link:href" files:html,js

```

Web recon tool to search for api leaks, and basically works like a web-grep! with builtin crawl, very fast!
