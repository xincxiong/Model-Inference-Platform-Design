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

export default function PlaygroundPage() {
  const [models, setModels] = useState<Model[]>([])
  const [selectedModel, setSelectedModel] = useState('')
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [temperature, setTemperature] = useState(0.7)
  const [maxTokens, setMaxTokens] = useState(2048)
  const [systemPrompt, setSystemPrompt] = useState('')
  const chatEndRef = useRef<HTMLDivElement>(null)
  const controllerRef = useRef<AbortController | null>(null)

  useEffect(() => {
    fetchModels()
      .then((data) => {
        const chatTypes = ['text-to-text', 'vision']
        const m = (data.models || []).filter((x: any) => chatTypes.includes(x.type))
        setModels(m)
        if (m.length > 0) setSelectedModel(m[0].id)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = () => {
    if (!input.trim() || isStreaming) return

    const userMsg: Message = { role: 'user', content: input.trim() }
    const allMessages = [...messages, userMsg]

    if (systemPrompt) {
      allMessages.unshift({ role: 'system', content: systemPrompt })
    }

    setMessages((prev) => [...prev, userMsg, { role: 'assistant', content: '' }])
    setInput('')
    setIsStreaming(true)

    controllerRef.current = streamChat(
      selectedModel,
      allMessages,
      { temperature, max_tokens: maxTokens },
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

  const exportCode = () => {
    const code = `from openai import OpenAI

client = OpenAI(
    base_url="${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/v1/",
    api_key="YOUR_API_KEY"
)

response = client.chat.completions.create(
    model="${selectedModel}",
    messages=${JSON.stringify(messages.filter(m => m.content), null, 4)},
    temperature=${temperature},
    max_tokens=${maxTokens},
    stream=True
)

for chunk in response:
    print(chunk.choices[0].delta.content or "", end="")`
    navigator.clipboard.writeText(code)
    alert('Python code copied to clipboard!')
  }

  return (
    <div className="flex gap-5 h-[calc(100vh-64px)]">
      {/* Chat area */}
      <div className="flex-1 flex flex-col min-w-0">
        <div className="flex items-center gap-3 mb-5">
          <h2 className="text-xl font-semibold text-[var(--text)]">Playground</h2>
          <div className="flex gap-2 ml-auto">
            <button onClick={handleClear} className="btn-outline text-sm">Clear</button>
            <button onClick={exportCode} className="btn-outline text-sm">Export Python</button>
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
            placeholder="输入消息..."
            className="flex-1 !rounded-full !px-5 !shadow-sm"
            disabled={isStreaming}
          />
          <button onClick={handleSend} className="btn-primary !rounded-full !px-6" disabled={isStreaming}>
            {isStreaming ? 'Generating...' : 'Send'}
          </button>
        </div>
      </div>

      {/* Params sidebar */}
      <div className="w-72 card h-fit space-y-5 shrink-0">
        <div>
          <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Model</label>
          <select value={selectedModel} onChange={(e) => setSelectedModel(e.target.value)}>
            {models.map((m) => (
              <option key={m.id} value={m.id}>{m.name}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
            Temperature: <span className="text-[var(--accent)] font-semibold">{temperature}</span>
          </label>
          <input
            type="range" min="0" max="2" step="0.1"
            value={temperature}
            onChange={(e) => setTemperature(parseFloat(e.target.value))}
            className="w-full"
          />
        </div>
        <div>
          <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Max Tokens</label>
          <input
            type="number" min="1" max="16384"
            value={maxTokens}
            onChange={(e) => setMaxTokens(parseInt(e.target.value) || 2048)}
          />
        </div>
        <div>
          <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">System Prompt</label>
          <textarea
            rows={3}
            value={systemPrompt}
            onChange={(e) => setSystemPrompt(e.target.value)}
            placeholder="You are a helpful assistant."
            className="text-sm"
          />
        </div>
      </div>
    </div>
  )
}
