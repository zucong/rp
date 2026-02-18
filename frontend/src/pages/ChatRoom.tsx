import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Send, Trash2, Edit2, X, Check, RefreshCw, Terminal, GitBranch } from 'lucide-react'
import LLMLogViewer from '../components/LLMLogViewer'
import DecisionTreeViewer from '../components/DecisionTreeViewer'

interface Message {
  id: number
  participant_id: number
  participant_name: string
  participant_avatar: string
  content: string
  created_at: string
  is_ai: boolean
  participant_type?: 'ai' | 'human'
}

interface Participant {
  id: number
  character_id: number
  character_name: string
  character_avatar: string
  participant_type: 'ai' | 'human'
  is_user: boolean
}

interface Room {
  id: number
  name: string
  setting: string
}

export default function ChatRoom() {
  const { id } = useParams<{ id: string }>()
  const roomId = parseInt(id || '0')
  const [room, setRoom] = useState<Room | null>(null)
  const [participants, setParticipants] = useState<Participant[]>([])
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [typingParticipants, setTypingParticipants] = useState<number[]>([])
  const [editingMessage, setEditingMessage] = useState<Message | null>(null)
  const [editContent, setEditContent] = useState('')
  const [viewingLogs, setViewingLogs] = useState<number | null>(null)
  const [viewingDecisions, setViewingDecisions] = useState<number | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const editTextareaRef = useRef<HTMLTextAreaElement | null>(null)
  const editContentRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (editTextareaRef.current) {
      const el = editTextareaRef.current
      el.style.height = 'auto'
      el.style.height = el.scrollHeight + 'px'
    }
  }, [editContent])

  useEffect(() => {
    fetchRoomData()
    fetchMessages()
    connectEventSource()
    return () => {
      eventSourceRef.current?.close()
    }
  }, [roomId])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const fetchRoomData = async () => {
    try {
      const [roomRes, participantsRes] = await Promise.all([
        fetch(`/api/rooms/${roomId}`),
        fetch(`/api/rooms/${roomId}/participants`),
      ])
      const roomData = await roomRes.json()
      const participantsData = await participantsRes.json()
      setRoom(roomData)
      setParticipants(participantsData || [])
    } catch (err) {
      console.error('Failed to fetch room data:', err)
    } finally {
      setLoading(false)
    }
  }

  const fetchMessages = async () => {
    try {
      const res = await fetch(`/api/rooms/${roomId}/messages`)
      const data = await res.json()
      setMessages(data || [])
    } catch (err) {
      console.error('Failed to fetch messages:', err)
    }
  }

  const connectEventSource = () => {
    eventSourceRef.current?.close()
    console.log('[SSE] Connecting to room', roomId)
    const es = new EventSource(`/api/rooms/${roomId}/events`)

    es.onopen = () => {
      console.log('[SSE] Connected')
    }

    es.onmessage = (event) => {
      console.log('[SSE] Received:', event.data)
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'message') {
          setMessages((prev) => [...prev, data.message])
          setTypingParticipants((prev) => prev.filter((id) => id !== data.message.participant_id))
        } else if (data.type === 'typing') {
          setTypingParticipants((prev) => [...prev, data.participant_id])
        } else if (data.type === 'message_edited') {
          setMessages((prev) =>
            prev.map((msg) =>
              msg.id === data.message_id ? { ...msg, content: data.content } : msg
            )
          )
        } else if (data.type === 'message_deleted') {
          setMessages((prev) => prev.filter((msg) => msg.id !== data.message_id))
        }
      } catch (err) {
        console.error('[SSE] Failed to parse message:', err)
      }
    }

    es.onerror = (err) => {
      console.error('[SSE] Error:', err)
    }

    eventSourceRef.current = es
  }

  const handleSend = async () => {
    if (!input.trim() || sending) return
    setSending(true)
    try {
      const res = await fetch(`/api/rooms/${roomId}/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: input }),
      })
      if (!res.ok) throw new Error('Failed to send')
      setInput('')
    } catch (err) {
      console.error('Failed to send message:', err)
    } finally {
      setSending(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleEditStart = (msg: Message) => {
    setEditingMessage(msg)
    setEditContent(msg.content)
    // Set content in next tick after DOM is rendered
    setTimeout(() => {
      if (editContentRef.current) {
        editContentRef.current.textContent = msg.content
      }
    }, 0)
  }

  const handleEditCancel = () => {
    setEditingMessage(null)
    setEditContent('')
  }

  const handleEditSave = async () => {
    if (!editingMessage || !editContent.trim()) return
    try {
      const res = await fetch(`/api/messages/${editingMessage.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: editContent }),
      })
      if (!res.ok) throw new Error('Failed to edit')
      setEditingMessage(null)
      setEditContent('')
    } catch (err) {
      console.error('Failed to edit message:', err)
      alert('Failed to edit message')
    }
  }

  const handleDelete = async (msgId: number) => {
    if (!confirm('Are you sure you want to delete this message?')) return
    try {
      const res = await fetch(`/api/messages/${msgId}`, {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error('Failed to delete')
    } catch (err) {
      console.error('Failed to delete message:', err)
      alert('Failed to delete message')
    }
  }

  const handleRegenerate = async () => {
    try {
      const res = await fetch(`/api/rooms/${roomId}/regenerate`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error('Failed to regenerate')
    } catch (err) {
      console.error('Failed to regenerate:', err)
      alert('Failed to regenerate AI response')
    }
  }

  const handleResetChat = async () => {
    if (!confirm('Are you sure you want to clear all chat history? This action cannot be undone.')) return
    try {
      const res = await fetch(`/api/rooms/${roomId}/messages`, {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error('Failed to reset chat')
      setMessages([])
    } catch (err) {
      console.error('Failed to reset chat:', err)
      alert('Failed to clear chat history')
    }
  }

  const currentUser = participants.find((p) => p.is_user)

  if (loading) return <div className="text-center py-8">Loading...</div>
  if (!room) return <div className="text-center py-8">Room not found</div>

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      <div className="border-b pb-4 mb-4">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold">{room.name}</h1>
          <button
            onClick={handleResetChat}
            className="px-3 py-1.5 text-sm text-destructive hover:bg-destructive/10 rounded-lg flex items-center gap-1.5"
          >
            <Trash2 className="h-4 w-4" />
            Clear History
          </button>
        </div>
        <p className="text-sm text-muted-foreground">{room.setting}</p>
        {currentUser && (
          <p className="text-sm text-primary mt-1">
            Current Identity: {currentUser.character_name}
          </p>
        )}
      </div>

      <div className="flex-1 overflow-y-auto space-y-4 pr-2">
        {(() => {
          const lastUserMsgId = [...messages].reverse().find(m => !m.is_ai)?.id
          return messages.map((msg) => {
            const isHuman = msg.participant_type === 'human' || !msg.is_ai
            const isLastUserMsg = msg.id === lastUserMsgId
            return (
            <div
              key={msg.id}
              className={`flex gap-3 group ${isHuman ? 'flex-row-reverse' : ''}`}
            >
              <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center text-lg flex-shrink-0">
                {msg.participant_avatar || msg.participant_name[0]}
              </div>
              <div className={`max-w-[70%] ${isHuman ? 'text-right' : ''}`}>
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-medium text-muted-foreground">
                    {msg.participant_name}
                  </span>
                </div>
                <div
                  className={`inline-block px-4 py-2 rounded-lg text-left ${
                    isHuman
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted'
                  }`}
                >
                  {editingMessage?.id === msg.id ? (
                    <>
                      <div
                        ref={editContentRef}
                        contentEditable
                        suppressContentEditableWarning
                        onInput={(e) => setEditContent(e.currentTarget.textContent || '')}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                            e.preventDefault()
                            handleEditSave()
                          }
                          if (e.key === 'Escape') handleEditCancel()
                        }}
                        className="outline-none whitespace-pre-wrap"
                        style={{ minHeight: '1.5em' }}
                        autoFocus
                      />
                      <div className={`flex gap-1 mt-2 ${isHuman ? 'justify-end' : ''}`}>
                        <button
                          onClick={handleEditSave}
                          className="p-1 rounded hover:bg-white/20"
                          title="Save"
                        >
                          <Check className="h-3.5 w-3.5" />
                        </button>
                        <button
                          onClick={handleEditCancel}
                          className="p-1 rounded hover:bg-white/20"
                          title="Cancel"
                        >
                          <X className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </>
                  ) : (
                    msg.content
                  )}
                </div>
                {!editingMessage && (
                  <div className={`flex gap-1 mt-1 ${isHuman ? 'justify-end' : ''}`}>
                    <button
                      onClick={() => handleEditStart(msg)}
                      className="p-1 rounded hover:bg-muted opacity-0 group-hover:opacity-100 transition-opacity"
                      title="Edit"
                    >
                      <Edit2 className="h-3 w-3" />
                    </button>
                    <button
                      onClick={() => handleDelete(msg.id)}
                      className="p-1 rounded hover:bg-muted text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                      title="Delete"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                    {isLastUserMsg && (
                      <button
                        onClick={handleRegenerate}
                        className="p-1 rounded hover:bg-muted opacity-0 group-hover:opacity-100 transition-opacity"
                        title="Regenerate AI Response"
                      >
                        <RefreshCw className="h-3 w-3" />
                      </button>
                    )}
                    {isHuman && (
                      <>
                        <button
                          onClick={() => setViewingDecisions(msg.id)}
                          className="p-1 rounded hover:bg-muted opacity-0 group-hover:opacity-100 transition-opacity"
                          title="View Decision Process"
                        >
                          <GitBranch className="h-3 w-3" />
                        </button>
                        <button
                          onClick={() => setViewingLogs(msg.id)}
                          className="p-1 rounded hover:bg-muted opacity-0 group-hover:opacity-100 transition-opacity"
                          title="View LLM Logs"
                        >
                          <Terminal className="h-3 w-3" />
                        </button>
                      </>
                    )}
                  </div>
                )}
              </div>
            </div>
          )
        })})()}

        {typingParticipants.length > 0 && (
          <div className="flex gap-3">
            {typingParticipants.map((pid) => {
              const p = participants.find((x) => x.id === pid)
              return (
                <div key={pid} className="flex items-center gap-2 text-sm text-muted-foreground">
                  <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center">
                    {p?.character_avatar || p?.character_name[0]}
                  </div>
                  <span>{p?.character_name} is typing...</span>
                </div>
              )
            })}
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="border-t pt-4 mt-4">
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a message... Use @CharacterName to force include, !CharacterName to force exclude"
            className="flex-1 px-4 py-2 border rounded-lg resize-none h-20"
          />
          <button
            onClick={handleSend}
            disabled={sending || !input.trim()}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50"
          >
            <Send className="h-5 w-5" />
          </button>
        </div>
      </div>

      {viewingLogs && (
        <LLMLogViewer
          messageId={viewingLogs}
          onClose={() => setViewingLogs(null)}
        />
      )}

      {viewingDecisions && (
        <DecisionTreeViewer
          messageId={viewingDecisions}
          onClose={() => setViewingDecisions(null)}
        />
      )}
    </div>
  )
}
