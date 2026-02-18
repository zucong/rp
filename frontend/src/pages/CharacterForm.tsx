import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'

interface CharacterFormData {
  name: string
  avatar: string
  prompt: string
  is_user_playable: boolean
  model_name: string
  temperature: number
  max_tokens: number
}

const defaultFormData: CharacterFormData = {
  name: '',
  avatar: '',
  prompt: '',
  is_user_playable: false,
  model_name: 'gpt-3.5-turbo',
  temperature: 0.7,
  max_tokens: 1000,
}

export default function CharacterForm() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEdit = !!id
  const [formData, setFormData] = useState<CharacterFormData>(defaultFormData)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (isEdit) {
      fetchCharacter()
    }
  }, [id])

  const fetchCharacter = async () => {
    try {
      const res = await fetch(`/api/characters/${id}`)
      const data = await res.json()
      setFormData(data)
    } catch (err) {
      console.error('Failed to fetch character:', err)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.name.trim()) {
      alert('Character name is required')
      return
    }

    setSaving(true)
    try {
      const url = isEdit ? `/api/characters/${id}` : '/api/characters'
      const method = isEdit ? 'PUT' : 'POST'
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      })
      if (!res.ok) throw new Error('Failed to save')
      navigate('/characters')
    } catch (err) {
      console.error('Failed to save character:', err)
      alert('Failed to save character')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">
        {isEdit ? 'Edit Character' : 'New Character'}
      </h1>

      <form onSubmit={handleSubmit} className="space-y-6">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Character Name *</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
              placeholder="e.g., Zhuge Liang"
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Avatar (Emoji or Character)</label>
            <input
              type="text"
              value={formData.avatar}
              onChange={(e) => setFormData({ ...formData, avatar: e.target.value })}
              className="w-full px-3 py-2 border rounded-md"
              placeholder="e.g., 🎭"
            />
          </div>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium">System Prompt *</label>
          <textarea
            value={formData.prompt}
            onChange={(e) => setFormData({ ...formData, prompt: e.target.value })}
            className="w-full px-3 py-2 border rounded-md h-32"
            placeholder="Describe this character's personality, background, speaking style..."
          />
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="playable"
            checked={formData.is_user_playable}
            onChange={(e) => setFormData({ ...formData, is_user_playable: e.target.checked })}
            className="w-4 h-4"
          />
          <label htmlFor="playable" className="text-sm">Allow users to play this character</label>
        </div>

        <div className="border-t pt-4">
          <h3 className="font-medium mb-4">Model Configuration</h3>
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Model Name</label>
              <input
                type="text"
                value={formData.model_name}
                onChange={(e) => setFormData({ ...formData, model_name: e.target.value })}
                className="w-full px-3 py-2 border rounded-md"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Temperature</label>
              <input
                type="number"
                step="0.1"
                min="0"
                max="2"
                value={formData.temperature}
                onChange={(e) => setFormData({ ...formData, temperature: parseFloat(e.target.value) })}
                className="w-full px-3 py-2 border rounded-md"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Max Tokens</label>
              <input
                type="number"
                value={formData.max_tokens}
                onChange={(e) => setFormData({ ...formData, max_tokens: parseInt(e.target.value) })}
                className="w-full px-3 py-2 border rounded-md"
              />
            </div>
          </div>
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
            onClick={() => navigate('/characters')}
            className="px-6 py-2 border rounded-md hover:bg-muted"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}
