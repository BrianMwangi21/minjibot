import { Routes, Route } from "react-router-dom"
import Home from "./pages/Home"
import Commands from "./pages/Commands"
import Login from "./pages/Login"
import SignUp from "./pages/SignUp"
import Dashboard from "./pages/Dashboard"
import GuildLogs from "./pages/GuildLogs"
import GuildSettings from "./pages/GuildSettings"
import ServerSetup from "./pages/ServerSetup"
import SetupGuide from "./pages/SetupGuide"
import Profile from "./pages/Profile"
import Diary from "./pages/Diary"
import { GuildLayout } from "./components/dashboard/GuildLayout"

export function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/commands" element={<Commands />} />
      <Route path="/login" element={<Login />} />
      <Route path="/signup" element={<SignUp />} />
      <Route path="/setup" element={<SetupGuide />} />
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/dashboard/guild/:guildId" element={<GuildLayout />}>
        <Route index element={<GuildLogs />} />
        <Route path="settings" element={<GuildSettings />} />
        <Route path="setup" element={<ServerSetup />} />
      </Route>
      <Route path="/dashboard/profile" element={<Profile />} />
      <Route path="/dashboard/diary" element={<Diary />} />
    </Routes>
  )
}

export default App
