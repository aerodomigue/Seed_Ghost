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
        <footer className="bg-dark-900 border-t border-dark-800 px-6 py-3">
          <div className="max-w-[90rem] mx-auto flex items-center justify-center gap-2 text-dark-500 text-xs">
            <a
              href="https://github.com/aerodomigue/Seed_Ghost"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 hover:text-dark-300 transition-colors"
            >
              <svg className="w-4 h-4" viewBox="0 0 16 16" fill="currentColor">
                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
              </svg>
              Seed_Ghost
            </a>
            <span className="text-dark-600">·</span>
            <span>{__APP_VERSION__}</span>
          </div>
        </footer>
      </div>
    </BrowserRouter>
  )
}
