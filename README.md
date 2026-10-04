# dijinlong-api-examples

Copy-paste examples for calling an OpenAI-compatible API.

Endpoints of this shape are all slightly different in the same annoying ways:
header names, streaming format, error payloads. Rather than re-deriving it every
time, this repo keeps working examples per language.

## Languages

| Language | Path |
|---|---|
| curl | `curl/` |
| Python | `python/` |
| Node.js | `node/` |
| Go | `go/` |

Each example covers:

- listing models
- a non-streaming chat completion
- a streaming chat completion
- handling an error response properly

## Usage

    export API_KEY="sk-..."
    bash curl/chat.sh

## Status

Early.
