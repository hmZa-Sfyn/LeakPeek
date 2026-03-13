#!/usr/bin/env python3

import argparse
import requests
from bs4 import BeautifulSoup
from urllib.parse import urljoin, urlparse
from concurrent.futures import ThreadPoolExecutor
import threading

visited = set()
lock = threading.Lock()


def valid_target(url, target):

    if target == "all":
        return True

    if target == "html":
        return url.endswith(".html") or "." not in url.split("/")[-1]

    if target == "js":
        return url.endswith(".js")

    if target == "css":
        return url.endswith(".css")

    return False


def extract_links(url):

    links = set()

    try:
        r = requests.get(url, timeout=10)
        r.raise_for_status()
    except Exception:
        return links

    soup = BeautifulSoup(r.text, "html.parser")

    for tag in soup.find_all("a", href=True):
        links.add(tag["href"])

    for tag in soup.find_all("script", src=True):
        links.add(tag["src"])

    for tag in soup.find_all("link", href=True):
        links.add(tag["href"])

    return links


def crawl(url, depth, max_depth, target, same_domain, base_domain, executor):

    if depth > max_depth:
        return

    with lock:
        if url in visited:
            return
        visited.add(url)

    print(url)

    links = extract_links(url)

    for link in links:

        absolute = urljoin(url, link)

        if same_domain:
            if urlparse(absolute).netloc != base_domain:
                continue

        if not valid_target(absolute, target):
            continue

        executor.submit(
            crawl,
            absolute,
            depth + 1,
            max_depth,
            target,
            same_domain,
            base_domain,
            executor,
        )


def main():

    parser = argparse.ArgumentParser(description="Recursive Web Crawler")

    parser.add_argument("url")
    parser.add_argument("--depth", type=int, default=2)
    parser.add_argument("--target", default="all",
                        choices=["all", "html", "js", "css"])
    parser.add_argument("--same-domain", action="store_true")

    args = parser.parse_args()

    base_domain = urlparse(args.url).netloc

    with ThreadPoolExecutor(max_workers=10) as executor:
        executor.submit(
            crawl,
            args.url,
            0,
            args.depth,
            args.target,
            args.same_domain,
            base_domain,
            executor,
        )


if __name__ == "__main__":
    main()