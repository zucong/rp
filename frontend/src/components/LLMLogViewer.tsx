import { useEffect, useState } from 'react'
import { X, Terminal, Clock, Hash, Thermometer, Maximize2 } from 'lucide-react'

interface LLMCallLog {
  id: number
  message_id: number
  room_id: number
  call_type: string
  model_name: string
  temperature: number
  max_tokens: number
  request_body: string
  response_body: string
  prompt_tokens: number
  completion_tokens: number
  latency_ms: number
  error_message: string
  created_at: string
}

interface LLMLogViewerProps {
  messageId: number
  onClose: () => void
}

export default function LLMLogViewer({ messageId, onClose }: LLMLogViewerProps) {
  const [logs, setLogs] = useState<LLMCallLog[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [expandedLog, setExpandedLog] = useState<number | null>(null)

  useEffect(() => {
    fetchLogs()
  }, [messageId])

  const fetchLogs = async () => {
    try {
      const res = await fetch(`/api/messages/${messageId}/llm-logs`)
      if (!res.ok) throw new Error('Failed to fetch logs')
      const data = await res.json()
      setLogs(data.logs || [])
    } catch (err) {
      setError('Failed to fetch logs')
    } finally {
      setLoading(false)
    }
  }

  const formatJson = (str: string) => {
    try {
      return JSON.stringify(JSON.parse(str), null, 2)
    } catch {
      return str
    }
  }

  const getCallTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      intent_analysis: 'Intent Analysis',
      fallback_selection: 'Fallback Selection',
      response_generation: 'Response Generation'
    }
    return labels[type] || type
  }

  if (loading) return <div className="p-4 text-center">Loading...</div>
  if (error) return <div className="p-4 text-center text-destructive">{error}</div>

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-4xl max-h-[80vh] flex flex-col m-4">
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center gap-2">
            <Terminal className="h-5 w-5" />
            <h2 className="text-lg font-semibold">LLM Call Logs</h2>
            <span className="text-sm text-muted-foreground">
              Message #{messageId} · {logs.length} calls
            </span>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-muted"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {logs.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">
              No log records
            </div>
          ) : (
            logs.map((log, index) => (
              <div
                key={log.id}
                className="border rounded-lg overflow-hidden"
              >
                <div
                  className="bg-muted/50 px-4 py-3 flex items-center justify-between cursor-pointer"
                  onClick={() => setExpandedLog(expandedLog === index ? null : index)}
                >
                  <div className="flex items-center gap-4">
                    <span className="font-medium">
                      #{index + 1} {getCallTypeLabel(log.call_type)}
                    </span>
                    <span className="text-sm text-muted-foreground">
                      {log.model_name}
                    </span>
                    {log.error_message && (
                      <span className="text-xs bg-destructive/10 text-destructive px-2 py-0.5 rounded">
                        Error
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3.5 w-3.5" />
                      {log.latency_ms}ms
                    </span>
                    <Maximize2 className="h-4 w-4" />
                  </div>
                </div>

                {expandedLog === index && (
                  <div className="p-4 space-y-4">
                    {/* Metadata */}
                    <div className="flex flex-wrap gap-4 text-sm">
                      <span className="flex items-center gap-1">
                        <Thermometer className="h-3.5 w-3.5" />
                        temperature: {log.temperature}
                      </span>
                      <span className="flex items-center gap-1">
                        <Hash className="h-3.5 w-3.5" />
                        max_tokens: {log.max_tokens}
                      </span>
                      {log.prompt_tokens > 0 && (
                        <span className="text-muted-foreground">
                          prompt_tokens: {log.prompt_tokens}
                        </span>
                      )}
                      {log.completion_tokens > 0 && (
                        <span className="text-muted-foreground">
                          completion_tokens: {log.completion_tokens}
                        </span>
                      )}
                    </div>

                    {/* Request */}
                    <div>
                      <h4 className="text-sm font-medium mb-2">Request</h4>
                      <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                        {formatJson(log.request_body)}
                      </pre>
                    </div>

                    {/* Response or Error */}
                    {log.error_message ? (
                      <div>
                        <h4 className="text-sm font-medium mb-2 text-destructive">Error</h4>
                        <pre className="bg-destructive/10 p-3 rounded text-xs text-destructive">
                          {log.error_message}
                        </pre>
                      </div>
                    ) : (
                      <div>
                        <h4 className="text-sm font-medium mb-2">Response</h4>
                        <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                          {log.response_body}
                        </pre>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
