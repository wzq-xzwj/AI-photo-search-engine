import { useNavigate } from 'react-router-dom'
import SearchBar from '../components/SearchBar'
import PhotoGrid from '../components/PhotoGrid'
import StatsPanel from '../components/StatsPanel'

interface HomeProps {
  selectedDir: string
}

export default function Home({ selectedDir }: HomeProps) {
  const navigate = useNavigate()

  const handleSearch = (query: string) => {
    if (query.trim()) {
      const params = new URLSearchParams({ q: query.trim() })
      if (selectedDir) params.set('dir', selectedDir)
      navigate(`/search?${params.toString()}`)
    }
  }

  return (
    <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative">

      {/* Search + Tags */}
      <section className="pt-12 md:pt-16 pb-6">
        <div className="max-w-xl mx-auto animate-fade-in">
          <SearchBar onSearch={handleSearch} placeholder="描述你想找的照片…比如「故宫的日落」「三亚沙滩」" />
        </div>
        <div className="flex flex-wrap justify-center gap-2 mt-4 animate-fade-in" style={{ animationDelay: '100ms' }}>
          {['👤 人像', '🏔️ 风景', '🍜 美食', '✈️ 旅行', '🏛️ 建筑', '🌃 夜景'].map(label => (
            <button
              key={label}
              onClick={() => {
                const params = new URLSearchParams({ q: label.replace(/^.\s/, '') })
                if (selectedDir) params.set('dir', selectedDir)
                navigate(`/search?${params.toString()}`)
              }}
              className="px-3.5 py-1.5 rounded-lg text-xs font-medium bg-white dark:bg-dark-card text-gray-600 dark:text-dark-muted hover:bg-purple-50 dark:hover:bg-purple-500/10 hover:text-purple-600 dark:hover:text-purple-400 border border-gray-200/60 dark:border-dark-border shadow-sm hover:shadow-md transition-all duration-200"
            >
              {label}
            </button>
          ))}
        </div>
      </section>

      {/* Stats */}
      <section className="mb-10 animate-fade-in" style={{ animationDelay: '150ms' }}>
        <StatsPanel selectedDir={selectedDir} />
      </section>

      {/* Photos */}
      <section className="pb-16">
        <PhotoGrid dir={selectedDir} />
      </section>
    </main>
  )
}
