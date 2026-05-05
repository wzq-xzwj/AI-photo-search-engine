import { useState, useEffect, ReactNode, Component } from 'react'
import { Routes, Route } from 'react-router-dom'
import Navbar from './components/Navbar'
import ChatPanel from './components/ChatPanel'
import Home from './pages/Home'
import Search from './pages/Search'
import Persons from './pages/Persons'

const SELECTED_DIR_KEY = 'photo-search-selected-dir'

class ErrorBoundary extends Component<{ children: ReactNode }, { hasError: boolean }> {
  state = { hasError: false }
  static getDerivedStateFromError() { return { hasError: true } }
  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-dark-bg">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">出错了</h2>
            <p className="text-gray-500 dark:text-dark-muted mb-6">页面遇到了一个问题</p>
            <button onClick={() => window.location.reload()} className="px-6 py-2.5 bg-purple-500 text-white rounded-xl hover:bg-purple-600 transition-colors">
              刷新页面
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}

function App() {
  const [darkMode, setDarkMode] = useState(() => {
    if (typeof window !== 'undefined') return window.matchMedia('(prefers-color-scheme: dark)').matches
    return false
  })

  const [selectedDir, setSelectedDir] = useState<string>('')

  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode)
  }, [darkMode])

  useEffect(() => {
    fetch('/api/v1/dirs')
      .then(res => res.json())
      .then(data => {
        const dirs = data.dirs || []
        const savedDir = localStorage.getItem(SELECTED_DIR_KEY)
        if (savedDir && dirs.includes(savedDir)) setSelectedDir(savedDir)
      })
      .catch(() => {})
  }, [])

  const handleDirChange = (dir: string) => {
    setSelectedDir(dir)
    try { dir ? localStorage.setItem(SELECTED_DIR_KEY, dir) : localStorage.removeItem(SELECTED_DIR_KEY) } catch {}
  }

  const toggleDarkMode = () => setDarkMode(!darkMode)

  return (
    <ErrorBoundary>
      <div className="min-h-screen bg-[#f5f4f0] dark:bg-dark-bg text-gray-900 dark:text-dark-text transition-colors duration-300 relative overflow-hidden">
        {/* 光晕装饰 */}
        <div className="fixed top-[-15%] left-[-8%] w-[600px] h-[600px] rounded-full bg-purple-300/20 dark:bg-purple-500/5 blur-[150px] pointer-events-none" />
        <div className="fixed bottom-[-10%] right-[-5%] w-[450px] h-[450px] rounded-full bg-indigo-200/20 dark:bg-indigo-500/5 blur-[120px] pointer-events-none" />
        <Navbar darkMode={darkMode} toggleDarkMode={toggleDarkMode} selectedDir={selectedDir} onDirChange={handleDirChange} onScanComplete={() => window.location.reload()} />
        <Routes>
          <Route path="/" element={<Home selectedDir={selectedDir} />} />
          <Route path="/search" element={<Search selectedDir={selectedDir} />} />
          <Route path="/persons" element={<Persons />} />
        </Routes>
        <ChatPanel />
      </div>
    </ErrorBoundary>
  )
}

export default App
