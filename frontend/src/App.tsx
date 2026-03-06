import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import Torrents from './pages/Torrents'
import Prowlarr from './pages/Prowlarr'
import Logs from './pages/Logs'
import Settings from './pages/Settings'

const navItems = [
  { path: '/', label: 'Dashboard' },
  { path: '/torrents', label: 'Torrents' },
  { path: '/prowlarr', label: 'Prowlarr' },
  { path: '/logs', label: 'Logs' },
  { path: '/settings', label: 'Settings' },
]

export default function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen flex flex-col">
        <nav className="bg-dark-900 border-b border-dark-800 px-6 py-3">
          <div className="max-w-[90rem] mx-auto flex items-center gap-8">
            <h1 className="text-xl font-bold text-ghost-400">
              SeedGhost
            </h1>
            <div className="flex gap-1">
              {navItems.map((item) => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  end={item.path === '/'}
                  className={({ isActive }) =>
                    `px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-dark-800 text-ghost-400'
                        : 'text-dark-400 hover:text-dark-200 hover:bg-dark-800/50'
                    }`
                  }
                >
                  {item.label}
                </NavLink>
              ))}
            </div>
          </div>
        </nav>
        <main className="flex-1 max-w-[90rem] mx-auto w-full p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/torrents" element={<Torrents />} />
            <Route path="/prowlarr" element={<Prowlarr />} />
            <Route path="/logs" element={<Logs />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}
