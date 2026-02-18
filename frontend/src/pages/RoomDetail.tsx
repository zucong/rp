import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, MessageSquare, Trash2, User } from 'lucide-react'

interface Room {
  id: number
  name: string
  description: string
  system_prompt: string
}

interface Character {
  id: number
  name: string
  avatar: string
  is_user_playable: boolean
}

interface Participant {
  id: number
  character_id: number
  character_name: string
  character_avatar: string
  participant_type: string
  is_user: boolean
}

export default function RoomDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [room, setRoom] = useState<Room | null>(null)
  const [participants, setParticipants] = useState<Participant[]>([])
  const [availableChars, setAvailableChars] = useState<Character[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
  }, [id])

  const fetchData = async () => {
    try {
      const [roomRes, participantsRes, charsRes] = await Promise.all([
        fetch(`/api/rooms/${id}`),
        fetch(`/api/rooms/${id}/participants`),
        fetch('/api/characters'),
      ])
      const roomData = await roomRes.json()
      const participantsData = await participantsRes.json()
      const charsData = await charsRes.json()

      setRoom(roomData)
      setParticipants(participantsData || [])

      // Filter out characters already in room
      const participantCharIds = new Set((participantsData || []).map((p: Participant) => p.character_id))
      setAvailableChars((charsData || []).filter((c: Character) => !participantCharIds.has(c.id)))
    } catch (err) {
      console.error('Failed to fetch data:', err)
    } finally {
      setLoading(false)
    }
  }

  const addParticipant = async (charId: number, isUser: boolean) => {
    try {
      await fetch(`/api/rooms/${id}/participants`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          character_id: charId,
          participant_type: isUser ? 'human' : 'ai',
          is_user: isUser,
        }),
      })
      fetchData()
    } catch (err) {
      console.error('Failed to add participant:', err)
    }
  }

  const removeParticipant = async (participantId: number) => {
    try {
      await fetch(`/api/rooms/${id}/participants/${participantId}`, {
        method: 'DELETE',
      })
      fetchData()
    } catch (err) {
      console.error('Failed to remove participant:', err)
    }
  }

  if (loading) return <div className="text-center py-8">Loading...</div>
  if (!room) return <div className="text-center py-8">Room not found</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/rooms" className="p-2 hover:bg-muted rounded-md">
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <h1 className="text-2xl font-bold">{room.name}</h1>
        <button
          onClick={() => navigate(`/rooms/${id}/chat`)}
          className="ml-auto inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
        >
          <MessageSquare className="h-4 w-4" />
          Enter Chat
        </button>
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        {/* Current Participants */}
        <div className="border rounded-lg p-4">
          <h2 className="font-semibold mb-4">Current Participants ({participants.length})</h2>
          {participants.length === 0 ? (
            <p className="text-muted-foreground">No participants yet. Add from the right panel.</p>
          ) : (
            <div className="space-y-2">
              {participants.map((p) => (
                <div
                  key={p.id}
                  className="flex items-center justify-between p-3 bg-muted rounded-md"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-sm">
                      {p.character_avatar || p.character_name[0]}
                    </div>
                    <div>
                      <span className="font-medium">{p.character_name}</span>
                      {p.is_user && (
                        <span className="ml-2 text-xs bg-primary text-primary-foreground px-2 py-0.5 rounded">
                          You
                        </span>
                      )}
                      {!p.is_user && (
                        <span className="ml-2 text-xs bg-secondary text-secondary-foreground px-2 py-0.5 rounded">
                          AI
                        </span>
                      )}
                    </div>
                  </div>
                  <button
                    onClick={() => removeParticipant(p.id)}
                    className="p-2 hover:bg-destructive/10 text-destructive rounded-md"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Available Characters */}
        <div className="border rounded-lg p-4">
          <h2 className="font-semibold mb-4">Available Characters</h2>
          {availableChars.length === 0 ? (
            <p className="text-muted-foreground">No characters available to add</p>
          ) : (
            <div className="space-y-2">
              {availableChars.map((char) => (
                <div
                  key={char.id}
                  className="flex items-center justify-between p-3 bg-muted rounded-md"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-sm">
                      {char.avatar || char.name[0]}
                    </div>
                    <span className="font-medium">{char.name}</span>
                    {char.is_user_playable && (
                      <span className="text-xs text-muted-foreground">(Playable)</span>
                    )}
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => addParticipant(char.id, false)}
                      className="px-3 py-1 text-sm bg-secondary hover:bg-secondary/80 rounded-md"
                    >
                      Add as AI
                    </button>
                    {char.is_user_playable && (
                      <button
                        onClick={() => addParticipant(char.id, true)}
                        className="px-3 py-1 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md"
                      >
                        <User className="h-3 w-3 inline mr-1" />
                        Play
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
