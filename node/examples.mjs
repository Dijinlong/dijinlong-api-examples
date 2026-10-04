// dijinlong-api-examples / node
// -----------------------------
// Four things you will actually need:
//
//   1. list models
//   2. a non-streaming chat completion
//   3. a streaming chat completion
//   4. a failed request, handled properly
//
// Uses the built-in fetch (Node 18+). No dependencies.
//
// Usage:
//     export API_KEY="sk-..."
//     node node/examples.mjs

const BASE = process.env.BASE_URL ?? "https://api.dijinlong.com/v1";
const MODEL = process.env.MODEL ?? "deepseek-v4-flash";
const API_KEY = process.env.API_KEY ?? "";

if (!API_KEY) {
  console.error("set API_KEY first (export API_KEY=sk-...)");
  process.exit(2);
}

const headers = {
  Authorization: `Bearer ${API_KEY}`,
  "Content-Type": "application/json",
};

function section(title) {
  console.log(`\n=== ${title} ===`);
}

// ---------------------------------------------------------------------------
section("1. list models");
try {
  const res = await fetch(`${BASE}/models`, { headers });
  const body = await res.json();
  const models = body.data ?? [];
  for (const m of models.slice(0, 20)) console.log("  -", m.id);
  console.log(`  (${models.length} total)`);
} catch (e) {
  console.log("  failed:", e.message);
}

// ---------------------------------------------------------------------------
section("2. chat completion (non-streaming)");
try {
  const res = await fetch(`${BASE}/chat/completions`, {
    method: "POST",
    headers,
    body: JSON.stringify({
      model: MODEL,
      messages: [{ role: "user", content: "Say hello in exactly three words." }],
    }),
  });
  const body = await res.json();
  console.log(" ", body.choices?.[0]?.message?.content ?? "(empty)");
  if (body.usage) console.log("  usage:", JSON.stringify(body.usage));
} catch (e) {
  console.log("  failed:", e.message);
}

// ---------------------------------------------------------------------------
section("3. chat completion (streaming)");
try {
  const res = await fetch(`${BASE}/chat/completions`, {
    method: "POST",
    headers: { ...headers, Accept: "text/event-stream" },
    body: JSON.stringify({
      model: MODEL,
      stream: true,
      messages: [{ role: "user", content: "Count from one to five." }],
    }),
  });

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  process.stdout.write("   ");

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    const lines = buffer.split("\n");
    buffer = lines.pop() ?? "";

    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed.startsWith("data:")) continue;
      const payload = trimmed.slice(5).trim();
      if (payload === "[DONE]") continue;
      try {
        const obj = JSON.parse(payload);
        const text = obj.choices?.[0]?.delta?.content ?? "";
        if (text) process.stdout.write(text);
      } catch {
        /* partial JSON across chunk boundaries -- ignore, next read completes it */
      }
    }
  }
  console.log();
} catch (e) {
  console.log("  failed:", e.message);
}

// ---------------------------------------------------------------------------
section("4. handling an error properly");
try {
  const res = await fetch(`${BASE}/chat/completions`, {
    method: "POST",
    headers,
    body: JSON.stringify({
      model: "definitely-not-a-real-model",
      messages: [{ role: "user", content: "hi" }],
    }),
  });

  const text = await res.text();
  console.log(`  HTTP ${res.status}`);

  if (res.ok) {
    console.log("  A 200 for a bogus model usually means the gateway silently");
    console.log("  substituted a default model. That is worth reporting.");
  } else {
    try {
      const parsed = JSON.parse(text);
      const msg = typeof parsed.error === "object" ? parsed.error?.message : parsed.error;
      console.log("  JSON error, message:", msg);
      console.log("  Good: machine-readable error, not an HTML page.");
    } catch {
      console.log("  Body is not JSON -- first 200 chars:");
      console.log("   ", text.slice(0, 200));
      console.log("  An HTML body from an API usually means a proxy or WAF answered,");
      console.log("  not the API itself.");
    }
  }
} catch (e) {
  console.log("  connection failed:", e.message);
}

console.log();
