#!/usr/bin/env python3
"""
Simple Gemini "hello" tester using multiple API keys from a file
Reads ./geminiapi.txt — one API key per line

Requirements:
    pip install google-generativeai
"""

import os
import sys
from pathlib import Path

try:
    import google.generativeai as genai
except ImportError:
    print("Error: google-generativeai package not found.")
    print("Please run:  pip install google-generativeai")
    sys.exit(1)


def load_api_keys(filepath="geminiapi.txt"):
    path = Path(filepath)
    if not path.is_file():
        print(f"Error: File not found → {path.resolve()}")
        sys.exit(1)

    keys = []
    with path.open(encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#"):
                keys.append(line)

    if not keys:
        print("Error: No API keys found in the file.")
        sys.exit(1)

    print(f"Loaded {len(keys)} API key(s) from {path.resolve()}")
    return keys


def try_say_hello(key: str) -> tuple[bool, str]:
    try:
        genai.configure(api_key=key)
        model = genai.GenerativeModel("gemini-1.5-flash")  # or gemini-1.5-pro, gemini-2.0-flash, etc.

        response = model.generate_content("hello")

        text = response.text.strip()
        if not text:
            return False, "Empty response"

        return True, text

    except Exception as e:
        err = str(e).strip()
        if len(err) > 120:
            err = err[:117] + "..."
        return False, err


def main():
    keys = load_api_keys()

    print("\nTrying to say 'hello' to Gemini...\n")

    working_key = None
    response_text = None

    for i, key in enumerate(keys, 1):
        print(f"  {i}/{len(keys)}  →  {key[:6]}...{key[-4:]}", end="  →  ", flush=True)

        success, result = try_say_hello(key)

        if success:
            print("OK ✓")
            working_key = key
            response_text = result
            break
        else:
            print(f"FAIL  ({result})")

    print("\n" + "─" * 60)

    if working_key:
        print("SUCCESS! Working API key found.")
        print(f"Key : {working_key[:8]}...{working_key[-6:]}")
        print("\nGemini says:\n")
        print(response_text)
        print()
    else:
        print("SORRY — None of the API keys worked.")
        print("Possible reasons:")
        print("  • all keys are invalid / expired / revoked")
        print("  • rate limit or quota exceeded on every key")
        print("  • network / Google API issue")
        print("  • IP blocked / region restriction")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nInterrupted.")
    except Exception as e:
        print("\nUnexpected error:", str(e))
