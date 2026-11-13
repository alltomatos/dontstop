#!/usr/bin/env python3
"""Vendored AgentRouter relay (MIT, fork of alltomatos/allcode).

Receives Anthropic Messages from the dontstop gateway and re-issues them through
the synchronous `anthropic` Python SDK, which is the only HTTP client shape that
passes AgentRouter's (agentrouter.org) connection-fingerprint WAF check.

The API key is taken from the incoming `x-api-key` header (the gateway keeps the
credential encrypted and applies tenant/auth/policies). The relay never logs the
payload or the key, and refuses requests without a key.

Env vars:
  AGENTROUTER_PORT      optional, default 7187
  AGENTROUTER_BIND      optional, default 127.0.0.1 (sidecar: 0.0.0.0)
  AGENTROUTER_UPSTREAM  optional, default https://agentrouter.org
  AGENTROUTER_API_KEY   optional fallback for standalone use (not used in Swarm)

Provenance/changes: see NOTICE.md. Upstream license: MIT (see LICENSE).
"""

import json
import os
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import anthropic

PORT = int(os.environ.get("AGENTROUTER_PORT", "7187"))
BIND = os.environ.get("AGENTROUTER_BIND", "127.0.0.1")
UPSTREAM = os.environ.get("AGENTROUTER_UPSTREAM", "https://agentrouter.org")
FALLBACK_KEY = (os.environ.get("AGENTROUTER_API_KEY") or "").strip()
# Bump together with the vendored manager.ts reference.
RELAY_VERSION = 3

# Fields AgentRouter's content filter rejects outright.
STRIP_KEYS = ("thinking", "output_config")
STRIP_BLOCK_TYPES = ("thinking", "redacted_thinking")

# SSE event types AgentRouter injects that OpenCode's parser rejects.
DROP_SSE_EVENTS = {"billing_summary"}

# Static stub — the real /v1/models on AgentRouter is blocked by the WAF.
# Availability is dynamic (402/503); this list is the catalog, not a health signal.
STUB_MODELS = [
    {"id": "gpt-6.1sol", "name": "gpt-6.1sol"},
    {"id": "gpt-6-astra", "name": "gpt-6-astra"},
    {"id": "claude-opus-5.5", "name": "claude-opus-5.5"},
    {"id": "gpt-5.6-sol", "name": "gpt-5.6-sol"},
    {"id": "claude-opus-5", "name": "claude-opus-5"},
    {"id": "claude-opus-4-8", "name": "claude-opus-4-8"},
    {"id": "deepseek-v4-flash", "name": "deepseek-v4-flash"},
    {"id": "glm-5.3", "name": "glm-5.3"},
]

_CLIENTS = {}


def client_for(api_key):
    key = (api_key or FALLBACK_KEY).strip()
    if not key:
        return None
    client = _CLIENTS.get(key)
    if client is None:
        client = anthropic.Anthropic(api_key=key, base_url=UPSTREAM)
        _CLIENTS[key] = client
    return client


def strip_cache_control(block):
    if "cache_control" not in block:
        return block
    return {k: v for k, v in block.items() if k != "cache_control"}


def clean_messages(messages):
    cleaned = []
    for message in messages:
        content = message.get("content")
        if isinstance(content, list):
            content = [
                strip_cache_control(block)
                for block in content
                if block.get("type") not in STRIP_BLOCK_TYPES
            ]
            message = {**message, "content": content}
        cleaned.append(message)
    return cleaned


def clean_payload(payload):
    payload = {k: v for k, v in payload.items() if k not in STRIP_KEYS}
    if isinstance(payload.get("messages"), list):
        payload["messages"] = clean_messages(payload["messages"])
    return payload


def describe_error(e):
    text = str(e)
    if "<html" in text.lower() or "<!doctype html" in text.lower():
        return "AgentRouter returned an HTML error page instead of a JSON response (likely blocked by their WAF or rate limiting) — try again in a moment."
    return text[:500]


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        # Never log headers or payloads (the key travels in x-api-key).
        try:
            sys.stderr.write("%s - %s\n" % (self.address_string(), fmt % args))
            sys.stderr.flush()
        except (OSError, ValueError):
            pass

    def _send_json(self, status, obj):
        body = json.dumps(obj).encode("utf-8")
        self.send_response(status)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/health":
            self._send_json(200, {"ok": True, "version": RELAY_VERSION})
            return
        if self.path in ("/v1/models", "/models"):
            self._send_json(200, {"data": STUB_MODELS})
            return
        self._send_json(404, {"error": "not found"})

    def do_POST(self):
        if self.path not in ("/v1/messages", "/messages"):
            self._send_json(404, {"error": "not found"})
            return
        client = client_for(self.headers.get("x-api-key", ""))
        if client is None:
            self._send_json(
                401,
                {"type": "error", "error": {"type": "authentication_error", "message": "missing x-api-key"}},
            )
            return
        length = int(self.headers.get("content-length", 0))
        raw = self.rfile.read(length) if length else b"{}"
        try:
            payload = clean_payload(json.loads(raw))
        except json.JSONDecodeError as e:
            self._send_json(400, {"error": f"invalid json: {e}"})
            return

        if payload.get("stream"):
            self._handle_stream(client, payload)
        else:
            self._handle_sync(client, payload)

    def _handle_sync(self, client, payload):
        payload = {**payload, "stream": False}
        try:
            result = client.messages.create(**payload)
        except anthropic.APIStatusError as e:
            body = e.body if isinstance(e.body, dict) else {
                "type": "error",
                "error": {"type": "api_error", "message": describe_error(e)},
            }
            self._send_json(e.status_code, body)
            return
        except Exception as e:
            self._send_json(502, {"type": "error", "error": {"type": "api_error", "message": describe_error(e)}})
            return
        self._send_json(200, result.model_dump(exclude_none=True))

    def _handle_stream(self, client, payload):
        payload = {**payload, "stream": True}
        self.send_response(200)
        self.send_header("content-type", "text/event-stream")
        self.send_header("cache-control", "no-cache")
        self.send_header("connection", "close")
        self.end_headers()
        self.close_connection = True

        try:
            stream = client.messages.create(**payload)
            for event in stream:
                event_type = getattr(event, "type", None)
                if event_type in DROP_SSE_EVENTS:
                    continue
                data = json.dumps(event.model_dump(exclude_none=True))
                chunk = f"event: {event_type}\ndata: {data}\n\n".encode("utf-8")
                try:
                    self.wfile.write(chunk)
                    self.wfile.flush()
                except (BrokenPipeError, ConnectionResetError):
                    return
        except Exception as e:
            try:
                err = json.dumps({"type": "error", "error": {"type": "api_error", "message": describe_error(e)}})
                self.wfile.write(f"event: error\ndata: {err}\n\n".encode("utf-8"))
            except (BrokenPipeError, ConnectionResetError):
                pass


def main():
    server = ThreadingHTTPServer((BIND, PORT), Handler)
    print(f"AgentRouter relay -> {UPSTREAM}")
    print(f"Listening on http://{BIND}:{PORT}")
    sys.stdout.flush()
    server.serve_forever()


if __name__ == "__main__":
    main()
