import os
import requests

url = "http://localhost:8080/v1/chat/completions"
api_key = os.getenv("ROUTERX_API_KEY", "")
if not api_key:
    raise SystemExit("Missing ROUTERX_API_KEY env var")

payload = {
    "model": "gemini-2.5-flash",
    "messages": [
        {"role": "user", "content": [{"type": "text", "text": "Hello from RouterX"}]}
    ]
}

headers = {
    "Authorization": f"Bearer {api_key}",
    "Content-Type": "application/json"
}

resp = requests.post(url, json=payload, headers=headers)
print(resp.status_code)
print(resp.text)
