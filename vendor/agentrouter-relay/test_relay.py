import io
import json
import os
import threading
import unittest
import urllib.error
import urllib.request
from contextlib import redirect_stderr
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class FakeUpstream(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_POST(self):
        length = int(self.headers.get("content-length", 0))
        if length:
            self.rfile.read(length)
        body = {
            "id": "msg_test",
            "type": "message",
            "role": "assistant",
            "model": "gpt-5.6-sol",
            "content": [{"type": "text", "text": "ok"}],
            "stop_reason": "end_turn",
            "stop_sequence": None,
            "usage": {"input_tokens": 1, "output_tokens": 1},
        }
        data = json.dumps(body).encode("utf-8")
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def serve(server):
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return thread


class RelayTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.upstream = ThreadingHTTPServer(("127.0.0.1", 0), FakeUpstream)
        serve(cls.upstream)
        os.environ["AGENTROUTER_UPSTREAM"] = f"http://127.0.0.1:{cls.upstream.server_address[1]}"
        os.environ["AGENTROUTER_BIND"] = "127.0.0.1"
        os.environ.pop("AGENTROUTER_API_KEY", None)
        global relay
        import relay as relay_module
        relay = relay_module
        cls.relay = relay_module
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), relay_module.Handler)
        serve(cls.server)
        cls.base = f"http://127.0.0.1:{cls.server.server_address[1]}"

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.upstream.shutdown()

    def test_health(self):
        with urllib.request.urlopen(f"{self.base}/health", timeout=5) as response:
            self.assertEqual(response.status, 200)
            body = json.loads(response.read())
            self.assertTrue(body["ok"])
            self.assertEqual(body["version"], self.relay.RELAY_VERSION)

    def test_rejects_without_key(self):
        request = urllib.request.Request(
            f"{self.base}/v1/messages",
            data=b"{}",
            headers={"content-type": "application/json"},
            method="POST",
        )
        with self.assertRaises(urllib.error.HTTPError) as caught:
            urllib.request.urlopen(request, timeout=5)
        self.assertEqual(caught.exception.code, 401)

    def test_chat_with_key(self):
        payload = json.dumps({
            "model": "gpt-5.6-sol",
            "max_tokens": 8,
            "messages": [{"role": "user", "content": "Reply with the single word: ok"}],
        }).encode("utf-8")
        request = urllib.request.Request(
            f"{self.base}/v1/messages",
            data=payload,
            headers={"content-type": "application/json", "x-api-key": "test-key-123"},
            method="POST",
        )
        with urllib.request.urlopen(request, timeout=10) as response:
            self.assertEqual(response.status, 200)
            body = json.loads(response.read())
            self.assertEqual(body["content"][0]["text"], "ok")

    def test_key_not_in_logs(self):
        payload = json.dumps({
            "model": "gpt-5.6-sol",
            "max_tokens": 8,
            "messages": [{"role": "user", "content": "hi"}],
        }).encode("utf-8")
        request = urllib.request.Request(
            f"{self.base}/v1/messages",
            data=payload,
            headers={"content-type": "application/json", "x-api-key": "super-secret-key"},
            method="POST",
        )
        buffer = io.StringIO()
        with redirect_stderr(buffer):
            with urllib.request.urlopen(request, timeout=10) as response:
                response.read()
        self.assertNotIn("super-secret-key", buffer.getvalue())


if __name__ == "__main__":
    unittest.main(verbosity=2)
