import { useState } from 'react'
import { Link } from 'react-router-dom'

interface NavbarProps {
  darkMode: boolean
  toggleDarkMode: () => void
}

export default function Navbar({ darkMode, toggleDarkMode }: NavbarProps) {
  const [scanDir, setScanDir] = useState('')
  const [scanning, setScanning] = useState(false)
  const [scanResult, setScanResult] = useState('')

  const handleScan = async () => {
    if (!scanDir.trim()) return
    setScanning(true)
    setScanResult('')
    try {
      const res = await fetch(`/api/v1/files/scan?dir=${encodeURIComponent(scanDir)}`, { method: 'POST' })
      const data = await res.json()
      setScanResult(`扫描完成！找到 ${data.files_scanned} 张照片`)
    } catch (err) {
      setScanResult('扫描失败')
    } finally {
      setScanning(false)
    }
  }

  return (
    <nav className="sticky top-0 z-50 glass dark:glass-dark">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link to="/" className="flex items-center space-x-3 group">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-primary-500 via-accent-500 to-secondary-500 flex items-center justify-center shadow-glow-sm group-hover:shadow-glow transition-shadow duration-300">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
            <span className="text-xl font-display font-bold text-gradient hidden sm:block">
              PhotoLens AI
            </span>
          </Link>

          {/* Actions */}
          <div className="flex items-center space-x-3">
            {/* Scan Button */}
            <div className="relative group">
              <button className="p-2.5 rounded-xl bg-primary-500 hover:bg-primary-600 text-white transition-colors duration-200">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                </svg>
              </button>
              <div className="absolute right-0 mt-2 w-80 bg-white dark:bg-dark-card rounded-xl shadow-lg p-4 hidden group-hover:block z-50">
                <p className="text-sm font-medium mb-2">扫描照片目录</p>
                <input
                  type="text"
                  value={scanDir}
                  onChange={(e) => setScanDir(e.target.value)}
                  placeholder="输入照片目录路径"
                  className="w-full p-2 text-sm border rounded-lg dark:bg-dark-surface dark:border-dark-border"
                />
                <button
                  onClick={handleScan}
                  disabled={scanning}
                  className="mt-2 w-full py-2 bg-primary-500 text-white rounded-lg text-sm hover:bg-primary-600 disabled:opacity-50"
                >
                  {scanning ? '扫描中...' : '开始扫描'}
                </button>
                {scanResult && (
                  <p className="mt-2 text-xs text-green-500">{scanResult}</p>
                )}
              </div>
            </div>

            {/* Theme Toggle */}
            <button
              onClick={toggleDarkMode}
              className="p-2.5 rounded-xl bg-gray-100 dark:bg-dark-surface hover:bg-gray-200 dark:hover:bg-dark-card transition-colors duration-200"
              aria-label="Toggle dark mode"
            >
              {darkMode ? (
                <svg className="w-5 h-5 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z" clipRule="evenodd" />
                </svg>
              ) : (
                <svg className="w-5 h-5 text-gray-600" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
                </svg>
              )}
            </button>
          </div>
        </div>
      </div>
    </nav>
  )
}
