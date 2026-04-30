import { useState, useEffect, Component, ReactNode } from 'react'
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
            <h2 className="text-2xl font-bold text-gray-900 dark:text-dark-text mb-2">出错了</h2>
            <p className="text-gray-500 dark:text-dark-muted mb-4">页面遇到了一个问题</p>
            <button onClick={() => window.location.reload()} className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600">
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
    if (typeof window !== 'undefined') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches
    }
    return false
  })

  const [selectedDir, setSelectedDir] = useState<string>('')

  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }, [darkMode])

  // 验证并恢复上次选择的目录
  useEffect(() => {
    fetch('/api/v1/dirs')
      .then(res => res.json())
      .then(data => {
        const dirs = data.dirs || []
        const savedDir = localStorage.getItem(SELECTED_DIR_KEY)
        if (savedDir && dirs.includes(savedDir)) {
          setSelectedDir(savedDir)
        }
      })
      .catch(() => {})
  }, [])

  const handleDirChange = (dir: string) => {
    setSelectedDir(dir)
    try {
      if (dir) {
        localStorage.setItem(SELECTED_DIR_KEY, dir)
      } else {
        localStorage.removeItem(SELECTED_DIR_KEY)
      }
    } catch {}
  }

  const toggleDarkMode = () => setDarkMode(!darkMode)

  return (
    <ErrorBoundary>
      <div className="min-h-screen bg-gray-50 dark:bg-dark-bg text-gray-900 dark:text-dark-text transition-colors duration-300">
        <Navbar darkMode={darkMode} toggleDarkMode={toggleDarkMode} selectedDir={selectedDir} onDirChange={handleDirChange} />
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
