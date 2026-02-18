import { useEffect, useState } from 'react'
import { X, GitBranch, ChevronDown, ChevronRight } from 'lucide-react'

interface DecisionStep {
  id: number
  message_id: number
  room_id: number
  step_order: number
  step_type: string
  input_data: string
  output_data: string
  llm_call_log_id: number
  reason: string
  created_at: string
}

interface DecisionTreeViewerProps {
  messageId: number
  onClose: () => void
}

export default function DecisionTreeViewer({ messageId, onClose }: DecisionTreeViewerProps) {
  const [steps, setSteps] = useState<DecisionStep[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set())

  useEffect(() => {
    fetchDecisions()
  }, [messageId])

  const fetchDecisions = async () => {
    try {
      const res = await fetch(`/api/messages/${messageId}/decisions`)
      if (!res.ok) throw new Error('Failed to fetch decisions')
      const data = await res.json()
      setSteps(data.decisions || [])
      // Expand all by default
      setExpandedSteps(new Set((data.decisions || []).map((s: DecisionStep) => s.id)))
    } catch (err) {
      setError('Failed to fetch decision process')
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

  const getStepTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      parse_mentions: 'Parse @ and ! commands',
      intent_analysis: 'Intent Analysis',
      fallback_selection: 'Fallback Selection',
      apply_force_include: 'Force Include Characters',
      apply_force_exclude: 'Force Exclude Characters',
      character_selection: 'Final Character Selection',
      response_generation: 'Generate Response'
    }
    return labels[type] || type
  }

  const getStepColor = (type: string) => {
    const colors: Record<string, string> = {
      parse_mentions: 'bg-blue-100 text-blue-800',
      intent_analysis: 'bg-purple-100 text-purple-800',
      fallback_selection: 'bg-orange-100 text-orange-800',
      apply_force_include: 'bg-green-100 text-green-800',
      apply_force_exclude: 'bg-red-100 text-red-800',
      character_selection: 'bg-indigo-100 text-indigo-800',
      response_generation: 'bg-gray-100 text-gray-800'
    }
    return colors[type] || 'bg-gray-100 text-gray-800'
  }

  const toggleStep = (id: number) => {
    const newExpanded = new Set(expandedSteps)
    if (newExpanded.has(id)) {
      newExpanded.delete(id)
    } else {
      newExpanded.add(id)
    }
    setExpandedSteps(newExpanded)
  }

  if (loading) return <div className="p-4 text-center">Loading...</div>
  if (error) return <div className="p-4 text-center text-destructive">{error}</div>

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-4xl max-h-[80vh] flex flex-col m-4">
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center gap-2">
            <GitBranch className="h-5 w-5" />
            <h2 className="text-lg font-semibold">Orchestrator Decision Process</h2>
            <span className="text-sm text-muted-foreground">
              Message #{messageId} · {steps.length} steps
            </span>
          </div>
          <button onClick={onClose} className="p-1 rounded hover:bg-muted">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          {steps.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">
              No decision records
            </div>
          ) : (
            <div className="space-y-4">
              {steps.map((step) => (
                <div key={step.id} className="border rounded-lg overflow-hidden">
                  <div
                    className="flex items-center gap-3 px-4 py-3 bg-muted/50 cursor-pointer"
                    onClick={() => toggleStep(step.id)}
                  >
                    <span className="text-sm font-medium text-muted-foreground w-8">
                      #{step.step_order}
                    </span>
                    <span className={`text-xs px-2 py-1 rounded ${getStepColor(step.step_type)}`}>
                      {getStepTypeLabel(step.step_type)}
                    </span>
                    <span className="flex-1 text-sm truncate">{step.reason}</span>
                    {expandedSteps.has(step.id) ? (
                      <ChevronDown className="h-4 w-4" />
                    ) : (
                      <ChevronRight className="h-4 w-4" />
                    )}
                  </div>

                  {expandedSteps.has(step.id) && (
                    <div className="p-4 space-y-4">
                      {/* Input */}
                      <div>
                        <h4 className="text-sm font-medium mb-2 text-muted-foreground">Input</h4>
                        <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                          {formatJson(step.input_data)}
                        </pre>
                      </div>

                      {/* Output */}
                      <div>
                        <h4 className="text-sm font-medium mb-2 text-muted-foreground">Output</h4>
                        <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                          {formatJson(step.output_data)}
                        </pre>
                      </div>

                      {/* Reason */}
                      <div className="text-sm text-muted-foreground border-t pt-2">
                        {step.reason}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
