import { useState, useRef, useEffect, useCallback, useMemo } from 'react'

interface Photo {
  id: string
  url: string
  thumbnail: string
  title: string
  date: string
  tags: string[]
  width: number
  height: number
}

interface PhotoGridProps {
  photos?: Photo[]
  loading?: boolean
}

export default function PhotoGrid({ photos: propPhotos, loading = false }: PhotoGridProps) {
  const [selectedPhoto, setSelectedPhoto] = useState<Photo | null>(null)
  const [visibleRange, setVisibleRange] = useState({ start: 0, end: 30 })
  const [realPhotos, setRealPhotos] = useState<Photo[]>([])
  const containerRef = useRef<HTMLDivElement>(null)

  // 如果没有传入照片，从后端获取真实照片
  useEffect(() => {
    if (!propPhotos) {
      fetch('/api/v1/photos?limit=30')
        .then(res => res.json())
        .then(data => {
          if (data.photos && data.photos.length > 0) {
            setRealPhotos(data.photos.map((p: any) => ({
              id: p.ID,
              url: `/api/v1/photo?path=${encodeURIComponent(p.Path)}`,
              thumbnail: `/api/v1/photo?path=${encodeURIComponent(p.Path)}`,
              title: p.Name,
              date: p.DateTime || '',
              tags: p.Tags || [],
              width: 400,
              height: 300,
            })))
          }
        })
        .catch(err => console.error('Failed to fetch photos:', err))
    }
  }, [propPhotos])

  const photos = useMemo(() => {
    if (propPhotos) return propPhotos
    if (realPhotos.length > 0) return realPhotos
    return [] // 不显示模拟数据
  }, [propPhotos, realPhotos])

  // Virtual scroll handler
  const handleScroll = useCallback(() => {
    if (!containerRef.current) return
    const { scrollTop } = containerRef.current
    const itemHeight = 280
    const start = Math.max(0, Math.floor(scrollTop / itemHeight) * 4 - 8)
    const end = Math.min(photos.length, start + 40)
    setVisibleRange({ start, end })
  }, [photos.length])

  useEffect(() => {
    const container = containerRef.current
    if (container) {
      container.addEventListener('scroll', handleScroll, { passive: true })
      return () => container.removeEventListener('scroll', handleScroll)
    }
  }, [handleScroll])

  if (loading) {
    return (
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 p-4">
        {Array.from({ length: 12 }).map((_, i) => (
          <div key={i} className="aspect-[4/3] rounded-xl bg-gray-200 dark:bg-dark-surface animate-pulse" />
        ))}
      </div>
    )
  }

  if (photos.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="w-20 h-20 rounded-2xl bg-gray-100 dark:bg-dark-surface flex items-center justify-center mb-6">
          <svg className="w-10 h-10 text-gray-400 dark:text-dark-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-dark-text mb-2">还没有照片</h3>
        <p className="text-sm text-gray-500 dark:text-dark-muted mb-6 max-w-sm">
          点击右上角的上传按钮，扫描你的照片目录开始使用
        </p>
      </div>
    )
  }

  return (
    <>
      <div
        ref={containerRef}
        className="h-[calc(100vh-200px)] overflow-y-auto scrollbar-thin"
      >
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 p-4">
          {photos.slice(visibleRange.start, visibleRange.end).map((photo, index) => (
            <div
              key={photo.id}
              className="group relative rounded-xl overflow-hidden cursor-pointer animate-fade-in bg-gray-100 dark:bg-dark-surface"
              style={{ animationDelay: `${(index % 12) * 50}ms` }}
              onClick={() => setSelectedPhoto(photo)}
            >
              <div className="aspect-[4/3] relative">
                <img
                  src={photo.thumbnail}
                  alt={photo.title}
                  className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
                  loading="lazy"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                <div className="absolute bottom-0 left-0 right-0 p-3 transform translate-y-full group-hover:translate-y-0 transition-transform duration-300">
                  <p className="text-white text-sm font-medium truncate">{photo.title}</p>
                  <p className="text-white/70 text-xs mt-0.5">{photo.date}</p>
                </div>
              </div>
              <div className="absolute top-2 right-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                {photo.tags.slice(0, 2).map((tag) => (
                  <span key={tag} className="px-2 py-0.5 text-xs rounded-full bg-primary-500/90 text-white backdrop-blur-sm">
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Photo Lightbox */}
      {selectedPhoto && (
        <div
          className="fixed inset-0 z-50 bg-black/90 flex items-center justify-center p-4 animate-fade-in"
          onClick={() => setSelectedPhoto(null)}
        >
          <div className="relative max-w-5xl max-h-[90vh] animate-slide-up" onClick={(e) => e.stopPropagation()}>
            <img
              src={selectedPhoto.url}
              alt={selectedPhoto.title}
              className="max-w-full max-h-[80vh] object-contain rounded-xl"
            />
            <div className="absolute bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black/80 to-transparent rounded-b-xl">
              <h3 className="text-white text-lg font-semibold">{selectedPhoto.title}</h3>
              <p className="text-white/70 text-sm mt-1">{selectedPhoto.date}</p>
              <div className="flex gap-2 mt-2">
                {selectedPhoto.tags.map((tag) => (
                  <span key={tag} className="px-2 py-0.5 text-xs rounded-full bg-primary-500/90 text-white">
                    {tag}
                  </span>
                ))}
              </div>
            </div>
            <button
              onClick={() => setSelectedPhoto(null)}
              className="absolute top-4 right-4 p-2 rounded-full bg-white/20 hover:bg-white/30 text-white backdrop-blur-sm transition-colors"
            >
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      )}
    </>
  )
}
