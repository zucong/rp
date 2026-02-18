import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, MessageSquare, Users } from 'lucide-react'

interface Room {
  id: number
  name: string
  description: string
  participant_count: number
  last_activity: string
}

export default function RoomList() {
  const [rooms, setRooms] = useState<Room[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchRooms()
  }, [])

  const fetchRooms = async () => {
    try {
      const res = await fetch('/api/rooms')
      const data = await res.json()
      setRooms(data || [])
    } catch (err) {
      console.error('Failed to fetch rooms:', err)
    } finally {
      setLoading(false)
    }
  }

  if (loading) return <div className="text-center py-8">Loading...</div>

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Rooms</h1>
        <Link
          to="/rooms/new"
          className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          New Room
        </Link>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {rooms.map((room) => (
          <Link
            key={room.id}
            to={`/rooms/${room.id}/detail`}
            className="border rounded-lg p-4 bg-card hover:shadow-md transition-shadow block"
          >
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-semibold text-lg">{room.name}</h3>
                <p className="text-sm text-muted-foreground mt-1">
                  {room.description || 'No description'}
                </p>
              </div>
              <MessageSquare className="h-5 w-5 text-muted-foreground" />
            </div>
            <div className="flex items-center gap-4 mt-4 text-sm text-muted-foreground">
              <span className="flex items-center gap-1">
                <Users className="h-4 w-4" />
                {room.participant_count} participants
              </span>
              {room.last_activity && (
                <span>Last active: {new Date(room.last_activity).toLocaleString()}</span>
              )}
            </div>
          </Link>
        ))}
      </div>

      {rooms.length === 0 && (
        <div className="text-center py-12 text-muted-foreground">
          No rooms yet. Click "New Room" to get started.
        </div>
      )}
    </div>
  )
}
