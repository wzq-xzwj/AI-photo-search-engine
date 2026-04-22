import { useNavigate } from 'react-router-dom'
import SearchBar from '../components/SearchBar'
import PhotoGrid from '../components/PhotoGrid'
import StatsPanel from '../components/StatsPanel'

export default function Home() {
  const navigate = useNavigate()

  const handleSearch = (query: string) => {
    if (query.trim()) {
      navigate(`/search?q=${encodeURIComponent(query.trim())}`)
    }
  }

  return (
    <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Hero Section */}
      <section className="text-center py-12 md:py-16">
        <div className="animate-slide-up">
          <h1 className="text-4xl md:text-6xl font-display font-bold mb-6">
            <span className="text-gradient">AI 驱动</span>
            <br />
            <span className="text-gray-900 dark:text-dark-text">照片搜索引擎</span>
          </h1>
          <p className="text-lg md:text-xl text-gray-600 dark:text-dark-muted max-w-2xl mx-auto mb-10">
            用自然语言描述你想要的照片，AI 帮你精准找到每一张珍贵回忆
          </p>
        </div>

        {/* Hero Search */}
        <div className="max-w-2xl mx-auto animate-fade-in" style={{ animationDelay: '200ms' }}>
          <SearchBar onSearch={handleSearch} placeholder="搜索照片... 尝试 '去年夏天在海边的照片'" />
        </div>

        {/* Quick Tags */}
        <div className="flex flex-wrap justify-center gap-3 mt-6 animate-fade-in" style={{ animationDelay: '400ms' }}>
          {['人像', '风景', '美食', '旅行', '宠物', '建筑'].map((tag) => (
            <button
              key={tag}
              onClick={() => navigate(`/search?q=${encodeURIComponent(tag)}`)}
              className="px-4 py-2 rounded-full text-sm font-medium bg-gray-100 dark:bg-dark-surface text-gray-700 dark:text-dark-muted hover:bg-primary-50 dark:hover:bg-primary-500/10 hover:text-primary-600 dark:hover:text-primary-400 border border-gray-200 dark:border-dark-border transition-all duration-200"
            >
              {tag}
            </button>
          ))}
        </div>
      </section>

      {/* Stats Section */}
      <section className="mb-12 animate-fade-in" style={{ animationDelay: '300ms' }}>
        <StatsPanel />
      </section>

      {/* Recent Photos */}
      <section className="mb-8">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-display font-bold text-gray-900 dark:text-dark-text">
              最近照片
            </h2>
            <p className="text-sm text-gray-500 dark:text-dark-muted mt-1">
              浏览你最近上传的照片
            </p>
          </div>
          <button
            onClick={() => navigate('/search')}
            className="text-sm text-primary-500 hover:text-primary-600 dark:text-primary-400 dark:hover:text-primary-300 font-medium flex items-center space-x-1 transition-colors"
          >
            <span>查看全部</span>
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
        <PhotoGrid />
      </section>
    </main>
  )
}
