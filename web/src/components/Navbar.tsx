import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'

interface NavbarProps {
  darkMode: boolean
  toggleDarkMode: () => void
  selectedDir: string
  onDirChange: (dir: string) => void
}

interface ScanTask {
  id: string
  status: string
  scanned: number
  total_files: number
  extracted: number
  current_file: string
  message: string
  error?: string
}

const RECENT_PATHS_KEY = 'photo-search-recent-paths'
const DEFAULT_PATH = '~/Pictures'

function getRecentPaths(): string[] {
  try {
    const raw = localStorage.getItem(RECENT_PATHS_KEY)
    if (raw) return JSON.parse(raw)
  } catch {}
  return []
}

function saveRecentPath(path: string) {
  if (!path || path === DEFAULT_PATH) return
  const existing = getRecentPaths()
  const next = [path, ...existing.filter((p) => p !== path)].slice(0, 5)
  try {
    localStorage.setItem(RECENT_PATHS_KEY, JSON.stringify(next))
  } catch {}
}

export default function Navbar({ darkMode, toggleDarkMode, selectedDir, onDirChange }: NavbarProps) {
  const [scanDir, setScanDir] = useState(DEFAULT_PATH)
  const [scanning, setScanning] = useState(false)
  const [scanResult, setScanResult] = useState('')
  const [scanTask, setScanTask] = useState<ScanTask | null>(null)
  const [scanMenuOpen, setScanMenuOpen] = useState(false)
  const [recentPaths, setRecentPaths] = useState<string[]>([])
  const [dirs, setDirs] = useState<string[]>([])
  const [dirMenuOpen, setDirMenuOpen] = useState(false)
  const scanMenuRef = useRef<HTMLDivElement>(null)
  const dirMenuRef = useRef<HTMLDivElement>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // 加载可用目录
  useEffect(() => {
    fetch('/api/v1/dirs')
      .then(res => res.json())
      .then(data => setDirs(data.dirs || []))
      .catch(() => {})
  }, [])

  useEffect(() => {
    setRecentPaths(getRecentPaths())
  }, [scanMenuOpen])

  // 点击外部关闭菜单
  useEffect(() => {
    if (!scanMenuOpen && !dirMenuOpen) return
    const onDocClick = (e: MouseEvent) => {
      if (scanMenuRef.current && !scanMenuRef.current.contains(e.target as Node)) {
        setScanMenuOpen(false)
      }
      if (dirMenuRef.current && !dirMenuRef.current.contains(e.target as Node)) {
        setDirMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [scanMenuOpen, dirMenuOpen])

  const clearScanInterval = () => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
  }

  useEffect(() => {
    return () => clearScanInterval()
  }, [])

  const handlePickFolder = async () => {
    try {
      const res = await fetch('/api/v1/system/pick-folder', { method: 'POST' })
      if (!res.ok) return
      const data = await res.json()
      if (data.path) {
        setScanDir(data.path)
      }
    } catch (err) {
      console.error('Pick folder failed:', err)
    }
  }

  const handleScan = async () => {
    if (!scanDir.trim()) return
    setScanning(true)
    setScanResult('')
    setScanTask(null)
    clearScanInterval()
    saveRecentPath(scanDir.trim())

    try {
      const res = await fetch(`/api/v1/files/scan?dir=${encodeURIComponent(scanDir.trim())}`, { method: 'POST' })
      const data = await res.json()
      if (!res.ok || data.error) {
        setScanResult(`启动扫描失败：${data.error || res.statusText}`)
        setScanning(false)
        return
      }
      const scanId = data.scan_id
      if (!scanId) {
        setScanResult('启动扫描失败：未返回扫描ID')
        setScanning(false)
        return
      }

      intervalRef.current = setInterval(async () => {
        try {
          const statusRes = await fetch(`/api/v1/files/scan/status?scan_id=${scanId}`)
          const task: ScanTask = await statusRes.json()
          setScanTask(task)

          if (task.status === 'completed' || task.status === 'failed') {
            clearScanInterval()
            setScanning(false)
            if (task.status === 'completed') {
              setScanResult(`扫描完成！共处理 ${task.total_files} 张照片`)
              // 扫描完成后刷新目录列表
              fetch('/api/v1/dirs')
                .then(res => res.json())
                .then(data => setDirs(data.dirs || []))
                .catch(() => {})
            } else {
              setScanResult(`扫描失败：${task.error || '未知错误'}`)
            }
          }
        } catch (err) {
          console.error('Scan status poll failed:', err)
        }
      }, 500)
    } catch (err) {
      setScanResult('扫描启动失败')
      setScanning(false)
    }
  }

  const progressPercent =
    scanTask && scanTask.total_files > 0
      ? Math.round(((scanTask.scanned + scanTask.extracted) / (scanTask.total_files * 2)) * 100)
      : 0

  // 截断长路径显示
  const truncatePath = (path: string, maxLen: number = 30) => {
    if (path.length <= maxLen) return path
    return '…' + path.slice(-(maxLen - 1))
  }

  return (
    <nav className="sticky top-0 z-50 glass dark:glass-dark">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <div className="flex items-center space-x-6">
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

            {/* Nav Links */}
            <Link
              to="/persons"
              className="flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-xl text-gray-600 dark:text-dark-muted hover:text-primary-600 dark:hover:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-500/10 transition-colors"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
              <span className="hidden sm:inline">人物</span>
            </Link>
          </div>

          {/* Actions */}
          <div className="flex items-center space-x-3">
            {/* Directory Selector */}
            {dirs.length > 0 && (
              <div className="relative" ref={dirMenuRef}>
                <button
                  onClick={() => setDirMenuOpen((v) => !v)}
                  className={`flex items-center gap-2 px-3 py-2 rounded-xl text-sm transition-colors duration-200 max-w-[240px] ${
                    selectedDir
                      ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
                      : 'bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-card'
                  }`}
                  title={selectedDir || '全部目录'}
                >
                  <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                  </svg>
                  <span className="truncate">{selectedDir ? truncatePath(selectedDir) : '全部目录'}</span>
                  <svg className="w-3 h-3 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </button>

                {dirMenuOpen && (
                  <div className="absolute right-0 mt-2 w-72 bg-white dark:bg-dark-card rounded-xl shadow-2xl border border-gray-100 dark:border-dark-border py-1 z-50 max-h-80 overflow-y-auto">
                    <button
                      onClick={() => { onDirChange(''); setDirMenuOpen(false) }}
                      className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${
                        !selectedDir
                          ? 'bg-primary-50 dark:bg-primary-500/10 text-primary-600 dark:text-primary-400'
                          : 'text-gray-700 dark:text-dark-text hover:bg-gray-50 dark:hover:bg-dark-surface'
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                        </svg>
                        全部目录
                      </div>
                    </button>
                    {dirs.map((dir) => (
                      <button
                        key={dir}
                        onClick={() => { onDirChange(dir); setDirMenuOpen(false) }}
                        className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${
                          selectedDir === dir
                            ? 'bg-primary-50 dark:bg-primary-500/10 text-primary-600 dark:text-primary-400'
                            : 'text-gray-700 dark:text-dark-text hover:bg-gray-50 dark:hover:bg-dark-surface'
                        }`}
                      >
                        <div className="flex items-center gap-2">
                          <svg className="w-4 h-4 flex-shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                          </svg>
                          <span className="truncate" title={dir}>{dir.split('/').pop() || dir}</span>
                        </div>
                        <div className="text-xs text-gray-400 dark:text-dark-muted ml-6 truncate">{dir}</div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Scan Button */}
            <div className="relative" ref={scanMenuRef}>
              <button
                onClick={() => setScanMenuOpen((v) => !v)}
                className={`p-2.5 rounded-xl text-white transition-colors duration-200 ${scanMenuOpen ? 'bg-primary-700' : 'bg-primary-500 hover:bg-primary-600'}`}
                aria-label="扫描照片"
                aria-expanded={scanMenuOpen}
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                </svg>
              </button>

              {scanMenuOpen && (
                <div className="absolute right-0 mt-2 w-80 sm:w-96 bg-white dark:bg-dark-card rounded-xl shadow-2xl border border-gray-100 dark:border-dark-border p-4 z-50">
                  <div className="flex items-center justify-between mb-3">
                    <p className="text-sm font-semibold text-gray-900 dark:text-dark-text">扫描照片目录</p>
                    <button
                      onClick={() => setScanMenuOpen(false)}
                      className="p-1 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-dark-text"
                      aria-label="关闭"
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>

                  <div className="flex gap-2 mb-2">
                    <input
                      type="text"
                      value={scanDir}
                      onChange={(e) => setScanDir(e.target.value)}
                      placeholder="输入照片目录路径，如 ~/Pictures"
                      className="flex-1 p-2 text-sm border rounded-lg dark:bg-dark-surface dark:border-dark-border focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                    <button
                      onClick={handlePickFolder}
                      className="px-3 py-2 text-sm rounded-lg bg-gray-100 dark:bg-dark-surface text-gray-700 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-border transition-colors whitespace-nowrap"
                      title="选择文件夹"
                    >
                      选择…
                    </button>
                  </div>

                  <div className="flex flex-wrap gap-2 mb-3">
                    <button
                      onClick={() => setScanDir(DEFAULT_PATH)}
                      className="px-2.5 py-1 text-xs rounded-md bg-gray-100 dark:bg-dark-surface text-gray-700 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-border transition-colors"
                    >
                      使用默认路径
                    </button>
                    {recentPaths.length > 0 && (
                      <select
                        value=""
                        onChange={(e) => {
                          if (e.target.value) setScanDir(e.target.value)
                        }}
                        className="px-2.5 py-1 text-xs rounded-md bg-gray-100 dark:bg-dark-surface text-gray-700 dark:text-dark-muted border-none cursor-pointer"
                      >
                        <option value="">最近路径…</option>
                        {recentPaths.map((p) => (
                          <option key={p} value={p}>
                            {p}
                          </option>
                        ))}
                      </select>
                    )}
                  </div>

                  <button
                    onClick={handleScan}
                    disabled={scanning || !scanDir.trim()}
                    className="w-full py-2 bg-primary-500 text-white rounded-lg text-sm font-medium hover:bg-primary-600 disabled:opacity-50 transition-colors"
                  >
                    {scanning ? '扫描中...' : '开始扫描'}
                  </button>

                  {/* 进度显示 */}
                  {scanning && (
                    <div className="mt-3">
                      {scanTask && scanTask.total_files > 0 && (
                        <>
                          <div className="flex justify-between text-xs text-gray-500 dark:text-dark-muted mb-1">
                            <span className="truncate max-w-[200px]">{scanTask.message}</span>
                            <span>{progressPercent}%</span>
                          </div>
                          <div className="w-full h-2 bg-gray-100 dark:bg-dark-surface rounded-full overflow-hidden">
                            <div
                              className="h-full bg-primary-500 transition-all duration-300"
                              style={{ width: `${Math.min(100, progressPercent)}%` }}
                            />
                          </div>
                        </>
                      )}
                      {scanTask && (
                        <>
                          <p className="mt-1 text-xs text-gray-400 dark:text-dark-muted truncate">
                            {scanTask.current_file || ''}
                          </p>
                          <p className="text-xs text-gray-400 dark:text-dark-muted">
                            已发现 {scanTask.scanned} 张 / 已提取 {scanTask.extracted} 张
                          </p>
                        </>
                      )}
                      {!scanTask && (
                        <p className="mt-1 text-xs text-gray-400 dark:text-dark-muted">正在启动扫描...</p>
                      )}
                    </div>
                  )}

                  {!scanning && scanResult && (
                    <p className={`mt-2 text-xs ${scanResult.startsWith('扫描失败') ? 'text-red-500' : 'text-green-500'}`}>
                      {scanResult}
                    </p>
                  )}
                </div>
              )}
            </div>

            {/* Theme Toggle */}
            <button
              onClick={toggleDarkMode}
              className="p-2.5 rounded-xl bg-gray-100 dark:bg-dark-surface hover:bg-gray-200 dark:hover:bg-dark-card transition-colors duration-200"
              aria-label="Toggle dark mode"
            >
              {darkMode ? (
                <svg className="w-5 h-5 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 01-1.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z" clipRule="evenodd" />
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
