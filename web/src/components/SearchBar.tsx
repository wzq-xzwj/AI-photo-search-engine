import { useState, useCallback } from 'react'

interface SearchBarProps {
  onSearch: (query: string) => void
  compact?: boolean
  placeholder?: string
}

export default function SearchBar({ onSearch, compact = false, placeholder }: SearchBarProps) {
  const [query, setQuery] = useState('')
  const [isFocused, setIsFocused] = useState(false)

  const handleSubmit = useCallback((e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim()) {
      onSearch(query.trim())
    }
  }, [query, onSearch])

  return (
    <form onSubmit={handleSubmit} className="w-full">
      <div className={`relative group ${isFocused ? 'ring-2 ring-primary-500/50 rounded-2xl' : ''} transition-all duration-300`}>
        <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
          <svg className={`w-5 h-5 transition-colors duration-200 ${isFocused ? 'text-primary-500' : 'text-gray-400 dark:text-dark-muted'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setIsFocused(false)}
          placeholder={placeholder || '搜索照片... 尝试 "去年夏天在海边的照片"'}
          className={`w-full ${compact ? 'py-2.5 pl-11 pr-4 text-sm' : 'py-4 pl-12 pr-6 text-base'} rounded-2xl border border-gray-200 dark:border-dark-border bg-white dark:bg-dark-surface text-gray-900 dark:text-dark-text focus:outline-none focus:border-primary-500 dark:focus:border-primary-400 transition-all duration-200 placeholder:text-gray-400 dark:placeholder:text-dark-muted shadow-sm`}
        />
        {query && (
          <button
            type="button"
            onClick={() => setQuery('')}
            className="absolute inset-y-0 right-0 pr-4 flex items-center"
          >
            <svg className="w-4 h-4 text-gray-400 hover:text-gray-600 dark:hover:text-dark-text transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>
    </form>
  )
}
