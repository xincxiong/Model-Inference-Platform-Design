'use client'

import { useState, useRef, useEffect } from 'react'
import { streamChat, fetchModels } from '@/lib/api'

interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
}

interface Model {
  id: string
  name: string
}

type PlaygroundMode = 'chat' | 'compare' | 'function' | 'structured'

type ResponseFormat =
  | { type: 'text' }
  | { type: 'json_object' }
  | { type: 'json_schema'; json_schema: { name: string; strict: boolean; schema: unknown } }

const DEFAULT_TOOLS = JSON.stringify(
  [
    {
      type: 'function',
      function: {
        name: 'get_weather',
        description: '获取指定城市的当前天气',
        parameters: {
          type: 'object',
          properties: {
            city: { type: 'string', description: '城市名称，如北京' },
            unit: { type: 'string', enum: ['celsius', 'fahrenheit'], description: '温度单位' },
          },
          required: ['city'],
        },
      },
    },
  ],
  null,
  2,
)

const DEFAULT_JSON_SCHEMA = JSON.stringify(
  {
    name: 'person',
    strict: true,
    schema: {
      type: 'object',
      properties: {
        name: { type: 'string', description: '姓名' },
        age: { type: 'integer', description: '年龄' },
        occupation: { type: 'string', description: '职业' },
      },
      required: ['name', 'age', 'occupation'],
      additionalProperties: false,
    },
  },
  null,
  2,
)

const MODE_LABELS: Record<PlaygroundMode, string> = {
  chat: '对话',
  compare: '模型对比',
  function: 'Function Calling',
  structured: '结构化输出',
}

export default function PlaygroundPage() {
  const [models, setModels] = useState<Model[]>([])
  const [mode, setMode] = useState<PlaygroundMode>('chat')

  // ── Chat / Function / Structured ──
  const [selectedModel, setSelectedModel] = useState('')
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [temperature, setTemperature] = useState(0.7)
  const [maxTokens, setMaxTokens] = useState(2048)
  const [topP, setTopP] = useState(1.0)
  const [systemPrompt, setSystemPrompt] = useState('')
  const chatEndRef = useRef<HTMLDivElement>(null)
  const controllerRef = useRef<{ abort: () => void } | null>(null)

  // ── Compare mode ──
  const [compareModelA, setCompareModelA] = useState('')
  const [compareModelB, setCompareModelB] = useState('')
  const [compareInput, setCompareInput] = useState('')
  const [compareSystemPrompt, setCompareSystemPrompt] = useState('')
  const [compareOutputA, setCompareOutputA] = useState('')
  const [compareOutputB, setCompareOutputB] = useState('')
  const [comparingA, setComparingA] = useState(false)
  const [comparingB, setComparingB] = useState(false)
  const controllerARef = useRef<{ abort: () => void } | null>(null)
  const controllerBRef = useRef<{ abort: () => void } | null>(null)

  // ── Function Calling ──
  const [toolsJSON, setToolsJSON] = useState(DEFAULT_TOOLS)
  const [toolsError, setToolsError] = useState('')
  const [toolChoice, setToolChoice] = useState<'auto' | 'none' | 'required'>('auto')
  const [functionMessages, setFunctionMessages] = useState<Message[]>([])
  const [functionInput, setFunctionInput] = useState('明天北京的天气怎么样？')
  const [functionResult, setFunctionResult] = useState('')
  const [functionStreaming, setFunctionStreaming] = useState(false)
  const functionControllerRef = useRef<{ abort: () => void } | null>(null)

  // ── Structured Output ──
  const [responseFormatMode, setResponseFormatMode] = useState<'json_object' | 'json_schema'>('json_schema')
  const [jsonSchemaJSON, setJsonSchemaJSON] = useState(DEFAULT_JSON_SCHEMA)
  const [schemaError, setSchemaError] = useState('')
  const [structuredInput, setStructuredInput] = useState('张伟，30岁，是一名软件工程师')
  const [structuredResult, setStructuredResult] = useState('')
  const [structuredStreaming, setStructuredStreaming] = useState(false)
  const structuredControllerRef = useRef<{ abort: () => void } | null>(null)

  useEffect(() => {
    fetchModels()
      .then((data) => {
        const chatTypes = ['text-to-text', 'vision']
        const m = (data.models || []).filter((x: Model & { type: string }) => chatTypes.includes(x.type))
        setModels(m)
        if (m.length > 0) {
          setSelectedModel(m[0].id)
          setCompareModelA(m[0].id)
          setCompareModelB(m.length > 1 ? m[1].id : m[0].id)
        }
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  // ── Chat ──────────────────────────────────────────────────────────────────

  const handleSend = () => {
    if (!input.trim() || isStreaming) return
    const userMsg: Message = { role: 'user', content: input.trim() }
    const allMessages: Message[] = []
    if (systemPrompt) allMessages.push({ role: 'system', content: systemPrompt })
    allMessages.push(...messages.filter(m => m.role !== 'system'), userMsg)

    setMessages((prev) => [...prev, userMsg, { role: 'assistant', content: '' }])
    setInput('')
    setIsStreaming(true)

    controllerRef.current = streamChat(
      selectedModel,
      allMessages,
      { temperature, max_tokens: maxTokens, top_p: topP },
      (text) => {
        setMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last.role === 'assistant') {
            updated[updated.length - 1] = { ...last, content: last.content + text }
          }
          return updated
        })
      },
      () => setIsStreaming(false),
      (err) => {
        setMessages((prev) => {
          const updated = [...prev]
          updated[updated.length - 1] = { role: 'assistant', content: `Error: ${err}` }
          return updated
        })
        setIsStreaming(false)
      },
    )
  }

  const handleClear = () => {
    setMessages([])
    controllerRef.current?.abort()
    setIsStreaming(false)
  }

  const exportCode = (lang: 'python' | 'javascript' | 'curl') => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
    const msgs = messages.filter((m) => m.content)
    let code = ''
    if (lang === 'python') {
      code = `from openai import OpenAI

client = OpenAI(
    base_url="${apiUrl}/v1/",
    api_key="YOUR_API_KEY"
)

response = client.chat.completions.create(
    model="${selectedModel}",
    messages=${JSON.stringify(msgs, null, 4)},
    temperature=${temperature},
    max_tokens=${maxTokens},
    stream=True
)

for chunk in response:
    print(chunk.choices[0].delta.content or "", end="")`
    } else if (lang === 'javascript') {
      code = `import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: '${apiUrl}/v1/',
  apiKey: 'YOUR_API_KEY',
});

const stream = await client.chat.completions.create({
  model: '${selectedModel}',
  messages: ${JSON.stringify(msgs, null, 2)},
  temperature: ${temperature},
  max_tokens: ${maxTokens},
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}`
    } else {
      code = `curl ${apiUrl}/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify({ model: selectedModel, messages: msgs, temperature, max_tokens: maxTokens, stream: true }, null, 2)}'`
    }
    navigator.clipboard.writeText(code)
  }

  // ── Compare ───────────────────────────────────────────────────────────────

  const handleCompare = () => {
    if (!compareInput.trim() || comparingA || comparingB) return
    setCompareOutputA('')
    setCompareOutputB('')
    setComparingA(true)
    setComparingB(true)

    const msgs: Message[] = []
    if (compareSystemPrompt) msgs.push({ role: 'system', content: compareSystemPrompt })
    msgs.push({ role: 'user', content: compareInput })

    controllerARef.current = streamChat(
      compareModelA, msgs, { temperature, max_tokens: maxTokens },
      (t) => setCompareOutputA((p) => p + t),
      () => setComparingA(false),
      (e) => { setCompareOutputA(`Error: ${e}`); setComparingA(false) },
    )
    controllerBRef.current = streamChat(
      compareModelB, msgs, { temperature, max_tokens: maxTokens },
      (t) => setCompareOutputB((p) => p + t),
      () => setComparingB(false),
      (e) => { setCompareOutputB(`Error: ${e}`); setComparingB(false) },
    )
  }

  const handleCompareStop = () => {
    controllerARef.current?.abort()
    controllerBRef.current?.abort()
    setComparingA(false)
    setComparingB(false)
  }

  // ── Function Calling ──────────────────────────────────────────────────────

  const handleFunctionSend = () => {
    setToolsError('')
    let parsedTools
    try {
      parsedTools = JSON.parse(toolsJSON)
    } catch {
      setToolsError('Tools JSON 格式错误')
      return
    }
    if (functionStreaming) return
    setFunctionResult('')
    setFunctionStreaming(true)
    const msgs: Message[] = []
    if (systemPrompt) msgs.push({ role: 'system', content: systemPrompt })
    msgs.push(...functionMessages, { role: 'user', content: functionInput })

    const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
    const key = typeof window !== 'undefined' ? (localStorage.getItem('api_key') || '') : ''
    const controller = new AbortController()
    functionControllerRef.current = controller

    fetch(`${API_URL}/v1/chat/completions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${key}` },
      signal: controller.signal,
      body: JSON.stringify({
        model: selectedModel,
        messages: msgs,
        tools: parsedTools,
        tool_choice: toolChoice,
        temperature,
        max_tokens: maxTokens,
        stream: false,
      }),
    })
      .then((r) => r.json())
      .then((data) => {
        setFunctionResult(JSON.stringify(data, null, 2))
        setFunctionMessages([...msgs, { role: 'assistant', content: data.choices?.[0]?.message?.content || '' }])
      })
      .catch((e) => { if (e.name !== 'AbortError') setFunctionResult(`Error: ${e.message}`) })
      .finally(() => setFunctionStreaming(false))
  }

  // ── Structured Output ─────────────────────────────────────────────────────

  const handleStructuredSend = () => {
    setSchemaError('')
    let responseFormat: ResponseFormat = { type: 'json_object' }
    if (responseFormatMode === 'json_schema') {
      try {
        const parsed = JSON.parse(jsonSchemaJSON)
        responseFormat = { type: 'json_schema', json_schema: parsed }
      } catch {
        setSchemaError('JSON Schema 格式错误')
        return
      }
    }
    if (structuredStreaming) return
    setStructuredResult('')
    setStructuredStreaming(true)

    const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
    const key = typeof window !== 'undefined' ? (localStorage.getItem('api_key') || '') : ''
    const controller = new AbortController()
    structuredControllerRef.current = controller

    const msgs: Message[] = []
    if (systemPrompt) msgs.push({ role: 'system', content: systemPrompt })
    msgs.push({ role: 'user', content: structuredInput })

    fetch(`${API_URL}/v1/chat/completions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${key}` },
      signal: controller.signal,
      body: JSON.stringify({
        model: selectedModel,
        messages: msgs,
        response_format: responseFormat,
        temperature,
        max_tokens: maxTokens,
        stream: false,
      }),
    })
      .then((r) => r.json())
      .then((data) => setStructuredResult(JSON.stringify(data, null, 2)))
      .catch((e) => { if (e.name !== 'AbortError') setStructuredResult(`Error: ${e.message}`) })
      .finally(() => setStructuredStreaming(false))
  }

  // ─────────────────────────────────────────────────────────────────────────

  return (
    <div className="flex flex-col h-[calc(100vh-64px)]">
      {/* Mode tabs */}
      <div className="flex gap-1 mb-5 border-b border-[var(--border-light)]">
        {(Object.keys(MODE_LABELS) as PlaygroundMode[]).map((m) => (
          <button
            key={m}
            type="button"
            onClick={() => setMode(m)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              mode === m
                ? 'border-[var(--accent)] text-[var(--accent)]'
                : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text)]'
            }`}
          >
            {MODE_LABELS[m]}
          </button>
        ))}
      </div>

      {/* ── Chat ─────────────────────────────────────────────────────────── */}
      {mode === 'chat' && (
        <div className="flex gap-5 flex-1 min-h-0">
          <div className="flex-1 flex flex-col min-w-0">
            <div className="flex items-center gap-3 mb-4">
              <h2 className="text-lg font-semibold text-[var(--text)]">对话</h2>
              <div className="flex gap-2 ml-auto">
                <button onClick={handleClear} className="btn-outline text-sm">清空</button>
                <div className="relative group">
                  <button className="btn-outline text-sm">导出代码 ▾</button>
                  <div className="absolute right-0 top-full mt-1 bg-white border border-[var(--border-light)] rounded-lg shadow-lg overflow-hidden z-10 hidden group-hover:block w-40">
                    {(['python', 'javascript', 'curl'] as const).map((lang) => (
                      <button key={lang} type="button" onClick={() => exportCode(lang)}
                        className="w-full text-left px-4 py-2 text-sm hover:bg-[var(--bg-hover)] text-[var(--text)]">
                        {lang === 'python' ? 'Python' : lang === 'javascript' ? 'JavaScript' : 'cURL'}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto mb-4 bg-white rounded-xl border border-[var(--border-light)] p-5">
              {messages.length === 0 && (
                <div className="h-full flex items-center justify-center text-[var(--text-muted)] text-sm">
                  <div className="text-center">
                    <svg className="mx-auto mb-3 text-[var(--border)]" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                    选择模型，输入消息开始对话
                  </div>
                </div>
              )}
              <div className="space-y-4">
                {messages.map((m, i) => (
                  <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                    <div className={`max-w-[75%] rounded-2xl px-4 py-3 text-sm whitespace-pre-wrap leading-relaxed ${
                      m.role === 'user'
                        ? 'bg-[var(--accent)] text-white'
                        : 'bg-[var(--bg-secondary)] text-[var(--text)] border border-[var(--border-light)]'
                    }`}>
                      {m.content || (isStreaming ? (
                        <span className="inline-flex gap-1">
                          <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{animationDelay: '0ms'}}/>
                          <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{animationDelay: '150ms'}}/>
                          <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{animationDelay: '300ms'}}/>
                        </span>
                      ) : '')}
                    </div>
                  </div>
                ))}
                <div ref={chatEndRef} />
              </div>
            </div>

            <div className="flex gap-3">
              <input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && handleSend()}
                placeholder="输入消息…"
                className="flex-1 !rounded-full !px-5 !shadow-sm"
                disabled={isStreaming}
              />
              <button onClick={handleSend} className="btn-primary !rounded-full !px-6" disabled={isStreaming}>
                {isStreaming ? '生成中…' : '发送'}
              </button>
            </div>
          </div>

          {/* Params sidebar */}
          <ParamsSidebar
            models={models} selectedModel={selectedModel} onModelChange={setSelectedModel}
            temperature={temperature} onTemperatureChange={setTemperature}
            maxTokens={maxTokens} onMaxTokensChange={setMaxTokens}
            topP={topP} onTopPChange={setTopP}
            systemPrompt={systemPrompt} onSystemPromptChange={setSystemPrompt}
          />
        </div>
      )}

      {/* ── Compare ──────────────────────────────────────────────────────── */}
      {mode === 'compare' && (
        <div className="flex gap-5 flex-1 min-h-0">
          <div className="flex-1 flex flex-col min-w-0">
            <div className="grid grid-cols-2 gap-4 mb-3">
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1">模型 A</label>
                <select value={compareModelA} onChange={(e) => setCompareModelA(e.target.value)}>
                  {models.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1">模型 B</label>
                <select value={compareModelB} onChange={(e) => setCompareModelB(e.target.value)}>
                  {models.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
                </select>
              </div>
            </div>

            <div className="mb-3">
              <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1">System Prompt（共用）</label>
              <textarea rows={2} value={compareSystemPrompt} onChange={(e) => setCompareSystemPrompt(e.target.value)}
                className="text-sm w-full" placeholder="可选系统提示词…" />
            </div>

            <div className="mb-3">
              <textarea rows={3} value={compareInput} onChange={(e) => setCompareInput(e.target.value)}
                className="text-sm w-full" placeholder="输入 Prompt，两个模型将同时生成回复…" />
            </div>

            <div className="flex gap-2 mb-4">
              <button type="button" onClick={handleCompare} disabled={comparingA || comparingB || !compareInput.trim()}
                className="btn-primary text-sm">
                {comparingA || comparingB ? '生成中…' : '同时发送两个模型'}
              </button>
              {(comparingA || comparingB) && (
                <button type="button" onClick={handleCompareStop} className="btn-outline text-sm">停止</button>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4 flex-1 min-h-0">
              {[
                { label: compareModelA, output: compareOutputA, streaming: comparingA },
                { label: compareModelB, output: compareOutputB, streaming: comparingB },
              ].map(({ label, output, streaming }, i) => {
                const modelName = models.find((m) => m.id === label)?.name || label
                return (
                  <div key={i} className="flex flex-col">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="text-xs font-medium text-[var(--text-secondary)]">{modelName}</span>
                      {streaming && <span className="w-1.5 h-1.5 rounded-full bg-[var(--accent)] animate-pulse" />}
                    </div>
                    <div className="flex-1 bg-white rounded-xl border border-[var(--border-light)] p-4 text-sm whitespace-pre-wrap overflow-y-auto text-[var(--text)] leading-relaxed min-h-32">
                      {output || (
                        <span className="text-[var(--text-muted)]">等待生成…</span>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          <ParamsSidebar
            models={models} selectedModel={compareModelA} onModelChange={() => {}}
            temperature={temperature} onTemperatureChange={setTemperature}
            maxTokens={maxTokens} onMaxTokensChange={setMaxTokens}
            topP={topP} onTopPChange={setTopP}
            systemPrompt={compareSystemPrompt} onSystemPromptChange={setCompareSystemPrompt}
            hideModelSelect
          />
        </div>
      )}

      {/* ── Function Calling ─────────────────────────────────────────────── */}
      {mode === 'function' && (
        <div className="flex gap-5 flex-1 min-h-0">
          <div className="flex-1 flex flex-col min-w-0 space-y-4">
            <div className="card">
              <h3 className="font-medium text-sm text-[var(--text)] mb-3">工具定义（JSON）</h3>
              <textarea
                rows={12}
                value={toolsJSON}
                onChange={(e) => setToolsJSON(e.target.value)}
                className="font-mono text-xs w-full"
              />
              {toolsError && <p className="text-[var(--danger)] text-xs mt-1">{toolsError}</p>}
              <div className="flex items-center gap-3 mt-3">
                <label className="text-xs font-medium text-[var(--text-secondary)]">tool_choice:</label>
                <select value={toolChoice} onChange={(e) => setToolChoice(e.target.value as 'auto' | 'none' | 'required')} className="text-sm">
                  <option value="auto">auto</option>
                  <option value="required">required</option>
                  <option value="none">none</option>
                </select>
              </div>
            </div>

            <div className="card">
              <h3 className="font-medium text-sm text-[var(--text)] mb-3">用户输入</h3>
              <div className="flex gap-2">
                <input value={functionInput} onChange={(e) => setFunctionInput(e.target.value)}
                  className="flex-1 text-sm" placeholder="输入会触发工具调用的问题…"
                  onKeyDown={(e) => e.key === 'Enter' && handleFunctionSend()} />
                <button type="button" onClick={handleFunctionSend} disabled={functionStreaming || !functionInput.trim()}
                  className="btn-primary text-sm">
                  {functionStreaming ? '调用中…' : '发送'}
                </button>
              </div>
            </div>

            {functionResult && (
              <div className="card flex-1">
                <h3 className="font-medium text-sm text-[var(--text)] mb-3">模型响应（完整 JSON）</h3>
                <pre className="text-xs font-mono text-[var(--text-secondary)] bg-[var(--bg-secondary)] rounded-lg p-4 overflow-auto max-h-96 whitespace-pre-wrap">
                  {functionResult}
                </pre>
              </div>
            )}
          </div>

          <ParamsSidebar
            models={models} selectedModel={selectedModel} onModelChange={setSelectedModel}
            temperature={temperature} onTemperatureChange={setTemperature}
            maxTokens={maxTokens} onMaxTokensChange={setMaxTokens}
            topP={topP} onTopPChange={setTopP}
            systemPrompt={systemPrompt} onSystemPromptChange={setSystemPrompt}
          />
        </div>
      )}

      {/* ── Structured Output ────────────────────────────────────────────── */}
      {mode === 'structured' && (
        <div className="flex gap-5 flex-1 min-h-0">
          <div className="flex-1 flex flex-col min-w-0 space-y-4">
            <div className="card">
              <h3 className="font-medium text-sm text-[var(--text)] mb-3">输出格式</h3>
              <div className="flex gap-3 mb-3">
                {(['json_object', 'json_schema'] as const).map((t) => (
                  <label key={t} className="flex items-center gap-2 cursor-pointer text-sm">
                    <input type="radio" name="format" value={t} checked={responseFormatMode === t}
                      onChange={() => setResponseFormatMode(t)} />
                    <span>{t === 'json_object' ? 'json_object（任意 JSON）' : 'json_schema（严格遵循 Schema）'}</span>
                  </label>
                ))}
              </div>

              {responseFormatMode === 'json_schema' && (
                <>
                  <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1">JSON Schema 定义</label>
                  <textarea
                    rows={12}
                    value={jsonSchemaJSON}
                    onChange={(e) => setJsonSchemaJSON(e.target.value)}
                    className="font-mono text-xs w-full"
                  />
                  {schemaError && <p className="text-[var(--danger)] text-xs mt-1">{schemaError}</p>}
                </>
              )}
            </div>

            <div className="card">
              <h3 className="font-medium text-sm text-[var(--text)] mb-3">用户输入</h3>
              <div className="flex gap-2">
                <input value={structuredInput} onChange={(e) => setStructuredInput(e.target.value)}
                  className="flex-1 text-sm" placeholder="描述需要提取或生成的信息…"
                  onKeyDown={(e) => e.key === 'Enter' && handleStructuredSend()} />
                <button type="button" onClick={handleStructuredSend} disabled={structuredStreaming || !structuredInput.trim()}
                  className="btn-primary text-sm">
                  {structuredStreaming ? '生成中…' : '生成'}
                </button>
              </div>
            </div>

            {structuredResult && (
              <div className="card flex-1">
                <h3 className="font-medium text-sm text-[var(--text)] mb-3">模型响应</h3>
                <pre className="text-xs font-mono text-[var(--text-secondary)] bg-[var(--bg-secondary)] rounded-lg p-4 overflow-auto max-h-96 whitespace-pre-wrap">
                  {structuredResult}
                </pre>
              </div>
            )}
          </div>

          <ParamsSidebar
            models={models} selectedModel={selectedModel} onModelChange={setSelectedModel}
            temperature={temperature} onTemperatureChange={setTemperature}
            maxTokens={maxTokens} onMaxTokensChange={setMaxTokens}
            topP={topP} onTopPChange={setTopP}
            systemPrompt={systemPrompt} onSystemPromptChange={setSystemPrompt}
          />
        </div>
      )}
    </div>
  )
}

// ── Shared params sidebar component ────────────────────────────────────────

interface ParamsSidebarProps {
  models: Model[]
  selectedModel: string
  onModelChange: (v: string) => void
  temperature: number
  onTemperatureChange: (v: number) => void
  maxTokens: number
  onMaxTokensChange: (v: number) => void
  topP: number
  onTopPChange: (v: number) => void
  systemPrompt: string
  onSystemPromptChange: (v: string) => void
  hideModelSelect?: boolean
}

function ParamsSidebar({
  models, selectedModel, onModelChange,
  temperature, onTemperatureChange,
  maxTokens, onMaxTokensChange,
  topP, onTopPChange,
  systemPrompt, onSystemPromptChange,
  hideModelSelect,
}: ParamsSidebarProps) {
  return (
    <div className="w-72 card h-fit space-y-5 shrink-0 overflow-y-auto">
      {!hideModelSelect && (
        <div>
          <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Model</label>
          <select value={selectedModel} onChange={(e) => onModelChange(e.target.value)}>
            {models.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
          </select>
        </div>
      )}
      <div>
        <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
          Temperature: <span className="text-[var(--accent)] font-semibold">{temperature}</span>
        </label>
        <input type="range" min="0" max="2" step="0.05"
          value={temperature} onChange={(e) => onTemperatureChange(parseFloat(e.target.value))}
          className="w-full" />
      </div>
      <div>
        <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
          Top P: <span className="text-[var(--accent)] font-semibold">{topP}</span>
        </label>
        <input type="range" min="0" max="1" step="0.05"
          value={topP} onChange={(e) => onTopPChange(parseFloat(e.target.value))}
          className="w-full" />
      </div>
      <div>
        <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Max Tokens</label>
        <input type="number" min="1" max="32768"
          value={maxTokens} onChange={(e) => onMaxTokensChange(parseInt(e.target.value) || 2048)} />
      </div>
      <div>
        <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">System Prompt</label>
        <textarea rows={4} value={systemPrompt} onChange={(e) => onSystemPromptChange(e.target.value)}
          placeholder="You are a helpful assistant." className="text-sm" />
      </div>
    </div>
  )
}
