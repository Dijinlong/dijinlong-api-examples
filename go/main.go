// dijinlong-api-examples / go
// ---------------------------
// Four things you will actually need:
//
//   1. list models
//   2. a non-streaming chat completion
//   3. a streaming chat completion
//   4. a failed request, handled properly
//
// Standard library only.
//
// Usage:
//     export API_KEY="sk-..."
//     go run go/main.go

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	base   = envOr("BASE_URL", "https://api.dijinlong.com/v1")
	model  = envOr("MODEL", "deepseek-v4-flash")
	apiKey = os.Getenv("API_KEY")
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func section(title string) {
	fmt.Printf("\n=== %s ===\n", title)
}

// post sends a chat-completions request. Caller closes the body.
func post(payload map[string]any, stream bool) (*http.Response, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, base+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	client := &http.Client{Timeout: 120 * time.Second}
	return client.Do(req)
}

func main() {
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "set API_KEY first (export API_KEY=sk-...)")
		os.Exit(2)
	}

	// -------------------------------------------------------------------
	section("1. list models")
	{
		req, _ := http.NewRequest(http.MethodGet, base+"/models", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
		if err != nil {
			fmt.Println("  failed:", err)
		} else {
			defer resp.Body.Close()
			var body struct {
				Data []struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				fmt.Println("  could not parse:", err)
			} else {
				for i, m := range body.Data {
					if i >= 20 {
						break
					}
					fmt.Println("  -", m.ID)
				}
				fmt.Printf("  (%d total)\n", len(body.Data))
			}
		}
	}

	// -------------------------------------------------------------------
	section("2. chat completion (non-streaming)")
	{
		resp, err := post(map[string]any{
			"model":    model,
			"messages": []map[string]string{{"role": "user", "content": "Say hello in exactly three words."}},
		}, false)
		if err != nil {
			fmt.Println("  failed:", err)
		} else {
			defer resp.Body.Close()
			var body struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				fmt.Println("  could not parse:", err)
			} else if len(body.Choices) > 0 {
				fmt.Println(" ", body.Choices[0].Message.Content)
			}
		}
	}

	// -------------------------------------------------------------------
	section("3. chat completion (streaming)")
	{
		resp, err := post(map[string]any{
			"model":    model,
			"stream":   true,
			"messages": []map[string]string{{"role": "user", "content": "Count from one to five."}},
		}, true)
		if err != nil {
			fmt.Println("  failed:", err)
		} else {
			defer resp.Body.Close()
			fmt.Print("   ")
			scanner := bufio.NewScanner(resp.Body)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if payload == "[DONE]" {
					break
				}
				var obj struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(payload), &obj); err == nil && len(obj.Choices) > 0 {
					fmt.Print(obj.Choices[0].Delta.Content)
				}
			}
			fmt.Println()
		}
	}

	// -------------------------------------------------------------------
	section("4. handling an error properly")
	{
		resp, err := post(map[string]any{
			"model":    "definitely-not-a-real-model",
			"messages": []map[string]string{{"role": "user", "content": "hi"}},
		}, false)
		if err != nil {
			fmt.Println("  connection failed:", err)
		} else {
			defer resp.Body.Close()
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			fmt.Printf("  HTTP %d\n", resp.StatusCode)

			if resp.StatusCode == http.StatusOK {
				fmt.Println("  A 200 for a bogus model usually means the gateway silently")
				fmt.Println("  substituted a default model. That is worth reporting.")
			} else {
				var parsed struct {
					Error any `json:"error"`
				}
				if json.Unmarshal(raw, &parsed) == nil {
					fmt.Printf("  JSON error: %v\n", parsed.Error)
					fmt.Println("  Good: machine-readable error, not an HTML page.")
				} else {
					s := string(raw)
					if len(s) > 200 {
						s = s[:200]
					}
					fmt.Println("  Body is not JSON -- first 200 chars:")
					fmt.Println("   ", s)
					fmt.Println("  An HTML body from an API usually means a proxy or WAF")
					fmt.Println("  answered, not the API itself.")
				}
			}
		}
	}

	fmt.Println()
}
