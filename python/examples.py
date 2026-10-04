#!/usr/bin/env python3
"""
dijinlong-api-examples / python
-------------------------------

Four things you will actually need, standard library only:

    1. list models
    2. a non-streaming chat completion
    3. a streaming chat completion
    4. a failed request, handled properly

Why standard library and not the openai package
----------------------------------------------
Because the point of these examples is to show the wire format. If you are
debugging why a gateway behaves differently from another gateway, an SDK that
hides the HTTP layer is the wrong tool.

Usage
-----
    export API_KEY="sk-..."
    python python/examples.py
"""

import json
import os
import sys
import urllib.error
import urllib.request

BASE = os.environ.get("BASE_URL", "https://api.dijinlong.com/v1")
MODEL = os.environ.get("MODEL", "deepseek-v4-flash")
API_KEY = os.environ.get("API_KEY", "")

if not API_KEY:
    print("set API_KEY first (export API_KEY=sk-...)", file=sys.stderr)
    sys.exit(2)


def request(path, payload=None, stream=False, timeout=60):
    url = BASE.rstrip("/") + path
    data = json.dumps(payload).encode() if payload is not None else None
    req = urllib.request.Request(url, data=data, method="POST" if data else "GET")
    req.add_header("Authorization", "Bearer " + API_KEY)
    req.add_header("Accept", "text/event-stream" if stream else "application/json")
    if data:
        req.add_header("Content-Type", "application/json")
    req.add_header("User-Agent", "dijinlong-api-examples")
    return urllib.request.urlopen(req, timeout=timeout)


def section(title):
    print("\n=== %s ===" % title)


# ---------------------------------------------------------------------------
section("1. list models")
try:
    with request("/models") as r:
        body = json.loads(r.read().decode())
    models = [m.get("id") for m in (body.get("data") or [])]
    for m in models[:20]:
        print("  -", m)
    print("  (%d total)" % len(models))
except urllib.error.HTTPError as e:
    print("  HTTP %s: %s" % (e.code, e.read().decode()[:200]))

# ---------------------------------------------------------------------------
section("2. chat completion (non-streaming)")
try:
    with request("/chat/completions", {
        "model": MODEL,
        "messages": [{"role": "user", "content": "Say hello in exactly three words."}],
    }) as r:
        body = json.loads(r.read().decode())
    print(" ", body["choices"][0]["message"]["content"])
    usage = body.get("usage") or {}
    if usage:
        print("  usage:", json.dumps(usage))
except urllib.error.HTTPError as e:
    print("  HTTP %s: %s" % (e.code, e.read().decode()[:200]))

# ---------------------------------------------------------------------------
section("3. chat completion (streaming)")
try:
    with request("/chat/completions", {
        "model": MODEL,
        "stream": True,
        "messages": [{"role": "user", "content": "Count from one to five."}],
    }, stream=True) as r:
        print("  ", end="")
        for raw in r:
            line = raw.decode("utf-8", "replace").strip()
            if not line.startswith("data:"):
                continue
            payload = line[5:].strip()
            if payload == "[DONE]":
                break
            try:
                obj = json.loads(payload)
            except json.JSONDecodeError:
                continue
            pieces = obj.get("choices") or []
            if not pieces:
                continue
            text = (pieces[0].get("delta") or {}).get("content") or ""
            sys.stdout.write(text)
            sys.stdout.flush()
    print()
except urllib.error.HTTPError as e:
    print("  HTTP %s: %s" % (e.code, e.read().decode()[:200]))

# ---------------------------------------------------------------------------
section("4. handling an error properly")
# Ask for a model that does not exist and look at what comes back.
try:
    with request("/chat/completions", {
        "model": "definitely-not-a-real-model",
        "messages": [{"role": "user", "content": "hi"}],
    }) as r:
        print("  HTTP %s (unexpected -- see note)" % r.status)
        print("  A 200 for a bogus model usually means the gateway silently")
        print("  substituted a default model. That is worth reporting.")
except urllib.error.HTTPError as e:
    raw = e.read().decode("utf-8", "replace")
    print("  HTTP %s" % e.code)
    try:
        parsed = json.loads(raw)
        msg = (parsed.get("error") or {}).get("message") if isinstance(parsed.get("error"), dict) else parsed.get("error")
        print("  JSON error, message:", msg)
        print("  Good: machine-readable error, not an HTML page.")
    except json.JSONDecodeError:
        print("  Body is not JSON -- first 200 chars:")
        print("   ", raw[:200])
        print("  An HTML body from an API usually means a proxy or WAF answered,")
        print("  not the API itself.")
except Exception as e:
    print("  connection failed:", e)

print()
