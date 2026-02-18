import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import CharacterList from './pages/CharacterList'
import CharacterForm from './pages/CharacterForm'
import RoomList from './pages/RoomList'
import RoomForm from './pages/RoomForm'
import RoomDetail from './pages/RoomDetail'
import ChatRoom from './pages/ChatRoom'
import Settings from './pages/Settings'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<RoomList />} />
        <Route path="/rooms" element={<RoomList />} />
        <Route path="/rooms/new" element={<RoomForm />} />
        <Route path="/rooms/:id/edit" element={<RoomForm />} />
        <Route path="/rooms/:id/detail" element={<RoomDetail />} />
        <Route path="/rooms/:id/chat" element={<ChatRoom />} />
        <Route path="/characters" element={<CharacterList />} />
        <Route path="/characters/new" element={<CharacterForm />} />
        <Route path="/characters/:id/edit" element={<CharacterForm />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Layout>
  )
}

export default App
