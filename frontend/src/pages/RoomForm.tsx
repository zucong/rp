import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'

interface RoomFormData {
  name: string
  description: string
  setting: string
}

const defaultFormData: RoomFormData = {
  name: '',
  description: '',
  setting: '',
}

export default function RoomForm() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEdit = !!id
  const [formData, setFormData] = useState<RoomFormData>(defaultFormData)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (isEdit) {
      fetchRoom()
    }
  }, [id])

  const fetchRoom = async () => {
    try {
      const res = await fetch(`/api/rooms/${id}`)
      const data = await res.json()
      setFormData(data)
    } catch (err) {
      console.error('Failed to fetch room:', err)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.name.trim()) {
      alert('Room name is required')
      return
    }

    setSaving(true)
    try {
      const url = isEdit ? `/api/rooms/${id}` : '/api/rooms'
      const method = isEdit ? 'PUT' : 'POST'
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      })
      if (!res.ok) throw new Error('Failed to save')
      navigate('/rooms')
    } catch (err) {
      console.error('Failed to save room:', err)
      alert('Failed to save room')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">
        {isEdit ? 'Edit Room' : 'New Room'}
      </h1>

      <form onSubmit={handleSubmit} className="space-y-6">
        <div className="space-y-2">
          <label className="text-sm font-medium">Room Name *</label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            className="w-full px-3 py-2 border rounded-md"
            placeholder="e.g., Heroes of the Three Kingdoms"
          />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium">Description</label>
          <input
            type="text"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            className="w-full px-3 py-2 border rounded-md"
            placeholder="Briefly describe the purpose of this room..."
          />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium">Room Setting</label>
          <textarea
            value={formData.setting}
            onChange={(e) => setFormData({ ...formData, setting: e.target.value })}
            className="w-full px-3 py-2 border rounded-md h-32"
            placeholder="Describe the background setting of this room. All characters will know this information..."
          />
        </div>

        <div className="flex gap-4 pt-4">
          <button
            type="submit"
            disabled={saving}
            className="px-6 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
          <button
            type="button"
            onClick={() => navigate('/rooms')}
            className="px-6 py-2 border rounded-md hover:bg-muted"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}
