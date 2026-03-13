#!/usr/bin/env python3

import argparse
import requests
import re
from concurrent.futures import ThreadPoolExecutor


def fetch_and_grep(url, pattern, regex, ignore_case):

    try:
        r = requests.get(url, timeout=10)
        text = r.text
    except Exception:
        return

    flags = re.IGNORECASE if ignore_case else 0

    if regex:
        compiled = re.compile(pattern, flags)

    for i, line in enumerate(text.splitlines(), 1):

        if regex:
            if compiled.search(line):
                print(f"{url}:{i}:{line.strip()}")
        else:
            if ignore_case:
                if pattern.lower() in line.lower():
                    print(f"{url}:{i}:{line.strip()}")
            else:
                if pattern in line:
                    print(f"{url}:{i}:{line.strip()}")


def load_urls(single, file):

    urls = []

    if single:
        urls.append(single)

    if file:
        with open(file) as f:
            for line in f:
                line = line.strip()
                if line:
                    urls.append(line)

    return urls


def main():

    parser = argparse.ArgumentParser(description="Remote URL Grep")

    parser.add_argument("-u", "--url")
    parser.add_argument("-l", "--list")
    parser.add_argument("-p", "--pattern", required=True)
    parser.add_argument("-r", "--regex", action="store_true")
    parser.add_argument("-i", "--ignore-case", action="store_true")

    args = parser.parse_args()

    urls = load_urls(args.url, args.list)

    with ThreadPoolExecutor(max_workers=10) as executor:

        for url in urls:
            executor.submit(
                fetch_and_grep,
                url,
                args.pattern,
                args.regex,
                args.ignore_case,
            )


if __name__ == "__main__":
    main()