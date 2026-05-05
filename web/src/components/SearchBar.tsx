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
    if (query.trim()) onSearch(query.trim())
  }, [query, onSearch])

  return (
    <form onSubmit={handleSubmit} className="w-full">
      <div className="relative">
        <div className={`
          flex items-center rounded-2xl overflow-hidden
          transition-all duration-300
          bg-white dark:bg-dark-card border
          ${isFocused
            ? 'border-purple-400 dark:border-purple-500 shadow-lg shadow-purple-500/10'
            : 'border-gray-200 dark:border-dark-border shadow-sm hover:shadow-md hover:border-gray-300 dark:hover:border-dark-border'
          }
        `}>
          <div className="pl-4 pr-2">
            <svg className={`w-5 h-5 transition-colors duration-200 ${isFocused ? 'text-purple-500' : 'text-gray-400'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
          <input
            type="text"
            value={query}
            onChange={e => setQuery(e.target.value)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setIsFocused(false)}
            placeholder={placeholder || '描述你想找的照片…'}
            className={`flex-1 outline-none bg-transparent text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500 ${compact ? 'py-2.5 text-sm' : 'py-3.5 text-base'}`}
          />
          {query && (
            <button type="button" onClick={() => setQuery('')} className="px-3 py-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
          <button
            type="submit"
            disabled={!query.trim()}
            className="m-1.5 px-4 py-2 rounded-xl bg-purple-500 text-white text-sm font-medium hover:bg-purple-600 disabled:opacity-40 disabled:cursor-not-allowed transition-all duration-200 shadow-sm shadow-purple-500/20"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </button>
        </div>
      </div>
    </form>
  )
}
