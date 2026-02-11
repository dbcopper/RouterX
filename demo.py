import requests

url = "http://localhost:8080/v1/chat/completions"
api_key = "user_key_39VSSxepGKcvWy68YF76eXerwBS"

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
