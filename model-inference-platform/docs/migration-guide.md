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

# 单轮调用
response = client.responses.create(
    model="deepseek-ai/DeepSeek-R1",
    instructions="你是一个有帮助的助手",
    input="Hello!"
)
print(response.output_text)

# 多轮对话 — previous_response_id 链式引用
response2 = client.responses.create(
    model="deepseek-ai/DeepSeek-R1",
    input="它和 RNN 的区别是什么？",
    previous_response_id=response.id,
    store=True
)
print(response2.output_text)
```

### Python (Embeddings)

```python
response = client.embeddings.create(
    model="BAAI/bge-m3",
    input=["你好世界", "Hello world"]
)
for item in response.data:
    print(f"[{item.index}] dim={len(item.embedding)}")
```

### Python (Image Generation)

```python
response = client.images.generate(
    model="black-forest-labs/FLUX.1-dev",
    prompt="A futuristic city with flying cars",
    size="1024x1024"
)
print(response.data[0].url)
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

### cURL (Embeddings)

```bash
curl https://your-platform.com/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-platform-key" \
  -d '{
    "model": "BAAI/bge-m3",
    "input": "你好世界"
  }'
```

### cURL (Rerank)

```bash
curl https://your-platform.com/v1/rerank \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-platform-key" \
  -d '{
    "model": "BAAI/bge-reranker-v2-m3",
    "query": "什么是 Transformer",
    "documents": ["Transformer 是一种注意力架构", "RNN 是循环网络", "CNN 用于图像识别"]
  }'
```

## API 兼容性

| Feature | Status | 说明 |
|---------|--------|------|
| Chat Completions API | ✅ Full | 含 streaming / tools / response_format 参数 |
| Responses API | ✅ Full | 含 store / previous_response_id / tools / text.format |
| Completions API (FIM) | ✅ Full | 文本补全，支持 suffix 参数 |
| Embeddings API | ✅ Full | 支持单条/批量输入 |
| Rerank API | ✅ Full | 支持 top_n 参数 |
| Images API | ✅ Full | 图像生成，支持 size / n 参数 |
| Streaming (SSE) | ✅ Full | Chat Completions + Responses API |
| Function Calling | ⬜ Phase 2 | tools 参数已声明，Mock 引擎待实现实际调用 |
| Structured Output | ⬜ Phase 2 | response_format 参数已声明，Mock 引擎待实现约束解码 |
| Models API | ✅ Full | 列表查询，含详细模型信息 |

> **注意**: Function Calling 和 Structured Output 的请求参数已在 API 层支持（可正常传入），但 Mock 引擎不执行实际的函数调用/结构化约束。接入 vLLM 后将完整支持。
