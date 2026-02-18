import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Pencil, Trash2 } from 'lucide-react'

interface Character {
  id: number
  name: string
  avatar: string
  prompt: string
  is_user_playable: boolean
  created_at: string
}

export default function CharacterList() {
  const [characters, setCharacters] = useState<Character[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchCharacters()
  }, [])

  const fetchCharacters = async () => {
    try {
      const res = await fetch('/api/characters')
      const data = await res.json()
      setCharacters(data || [])
    } catch (err) {
      console.error('Failed to fetch characters:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this character?')) return
    try {
      await fetch(`/api/characters/${id}`, { method: 'DELETE' })
      setCharacters(characters.filter(c => c.id !== id))
    } catch (err) {
      console.error('Failed to delete character:', err)
    }
  }

  if (loading) return <div className="text-center py-8">Loading...</div>

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Characters</h1>
        <Link
          to="/characters/new"
          className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          New Character
        </Link>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {characters.map((char) => (
          <div
            key={char.id}
            className="border rounded-lg p-4 bg-card hover:shadow-md transition-shadow"
          >
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center text-lg">
                  {char.avatar || char.name[0]}
                </div>
                <div>
                  <h3 className="font-semibold">{char.name}</h3>
                  {char.is_user_playable && (
                    <span className="text-xs text-muted-foreground">Playable</span>
                  )}
                </div>
              </div>
              <div className="flex gap-2">
                <Link
                  to={`/characters/${char.id}/edit`}
                  className="p-2 hover:bg-muted rounded-md"
                >
                  <Pencil className="h-4 w-4" />
                </Link>
                <button
                  onClick={() => handleDelete(char.id)}
                  className="p-2 hover:bg-destructive/10 text-destructive rounded-md"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
            <p className="mt-3 text-sm text-muted-foreground line-clamp-2">
              {char.prompt}
            </p>
          </div>
        ))}
      </div>

      {characters.length === 0 && (
        <div className="text-center py-12 text-muted-foreground">
          No characters yet. Click "New Character" to get started.
        </div>
      )}
    </div>
  )
}
