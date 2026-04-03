# Migration Guide — 从 OpenAI 迁移到推理平台

## 3 步完成迁移

### Step 1: 修改 base_url

将 OpenAI API 地址替换为平台地址，其他代码无需改动。

### Step 2: 替换 api_key

使用平台控制台创建的 API Key 替换 OpenAI Key。

### Step 3: 指定模型名

将模型名替换为平台支持的模型标识（如 `deepseek-ai/DeepSeek-R1`）。

---

## 代码示例

### Python (Chat Completions)

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://your-platform.com/v1/",
    api_key="sk-your-platform-key"
)

response = client.chat.completions.create(
    model="deepseek-ai/DeepSeek-R1",
    messages=[
        {"role": "system", "content": "你是一个有帮助的助手"},
        {"role": "user", "content": "Hello!"}
    ],
    stream=True
)

for chunk in response:
    print(chunk.choices[0].delta.content or "", end="")
```

### Python (Responses API)

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://your-platform.com/v1/",
    api_key="sk-your-platform-key"
)

response = client.responses.create(
    model="deepseek-ai/DeepSeek-R1",
    instructions="你是一个有帮助的助手",
    input="Hello!"
)
print(response.output_text)
```

### JavaScript

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'https://your-platform.com/v1/',
  apiKey: 'sk-your-platform-key',
});

const response = await client.chat.completions.create({
  model: 'deepseek-ai/DeepSeek-R1',
  messages: [{ role: 'user', content: 'Hello!' }],
});
console.log(response.choices[0].message.content);
```

### cURL

```bash
curl https://your-platform.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-platform-key" \
  -d '{
    "model": "deepseek-ai/DeepSeek-R1",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## API 兼容性

| Feature | Status |
|---------|--------|
| Chat Completions API | ✅ Full |
| Responses API | ✅ Full |
| Completions API (FIM) | ✅ Full |
| Embeddings API | ✅ Full |
| Streaming (SSE) | ✅ Full |
| Function Calling | ✅ Full |
| Structured Output | ✅ Full |
