import { useState, useEffect } from 'react'

interface Config {
  api_endpoint: string
  api_key: string
  default_model: string
}

const defaultConfig: Config = {
  api_endpoint: 'https://api.openai.com/v1',
  api_key: '',
  default_model: 'gpt-3.5-turbo',
}

export default function Settings() {
  const [config, setConfig] = useState<Config>(defaultConfig)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    fetchConfig()
  }, [])

  const fetchConfig = async () => {
    try {
      const res = await fetch('/api/config')
      const data = await res.json()
      setConfig({ ...defaultConfig, ...data })
    } catch (err) {
      console.error('Failed to fetch config:', err)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const res = await fetch('/api/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
      })
      if (!res.ok) throw new Error('Failed to save')
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      console.error('Failed to save config:', err)
      alert('Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Settings</h1>

      <div className="space-y-6">
        <div className="space-y-2">
          <label className="text-sm font-medium">LLM API Endpoint</label>
          <input
            type="text"
            value={config.api_endpoint}
            onChange={(e) => setConfig({ ...config, api_endpoint: e.target.value })}
            className="w-full px-3 py-2 border rounded-md"
            placeholder="https://api.openai.com/v1"
          />
          <p className="text-xs text-muted-foreground">
            Supports OpenAI-compatible API endpoints, such as OpenAI, Azure, local Ollama, etc.
          </p>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium">API Key</label>
          <input
            type="password"
            value={config.api_key}
            onChange={(e) => setConfig({ ...config, api_key: e.target.value })}
            className="w-full px-3 py-2 border rounded-md"
            placeholder="sk-..."
          />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium">Default Model</label>
          <input
            type="text"
            value={config.default_model}
            onChange={(e) => setConfig({ ...config, default_model: e.target.value })}
            className="w-full px-3 py-2 border rounded-md"
            placeholder="gpt-3.5-turbo"
          />
        </div>

        <div className="pt-4">
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-6 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
          >
            {saving ? 'Saving...' : saved ? 'Saved!' : 'Save Settings'}
          </button>
        </div>
      </div>
    </div>
  )
}
