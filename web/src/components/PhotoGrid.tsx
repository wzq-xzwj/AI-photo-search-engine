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
  cameraMake?: string
  cameraModel?: string
}

interface PhotoGridProps {
  photos?: Photo[]
  loading?: boolean
  dir?: string
  hasMore?: boolean
  onLoadMore?: () => void
}

function getColumns() {
  if (typeof window === 'undefined') return 4
  if (window.innerWidth >= 1024) return 4
  if (window.innerWidth >= 768) return 3
  return 2
}

const GAP = 16
const ROW_HEIGHT_ESTIMATE = 220
const PAGE_SIZE = 30
const OVERSCAN_ROWS = 3

// 使用 IntersectionObserver 进行图片懒加载
function useLazyImage(src: string, placeholder: string) {
  const [imgSrc, setImgSrc] = useState(placeholder)
  const imgRef = useRef<HTMLImageElement>(null)

  useEffect(() => {
    const img = imgRef.current
    if (!img) return

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setImgSrc(src)
            observer.disconnect()
          }
        })
      },
      { rootMargin: '200px' }
    )

    observer.observe(img)
    return () => observer.disconnect()
  }, [src])

  return { imgSrc, imgRef }
}

// 单个照片卡片组件 - 独立减少重渲染
const PhotoCard = React.memo(function PhotoCard({
  photo,
  index,
  onClick,
}: {
  photo: Photo
  index: number
  onClick: (photo: Photo) => void
}) {
  const { imgSrc, imgRef } = useLazyImage(photo.thumbnail, '')

  return (
    <div
      className="group relative rounded-xl overflow-hidden cursor-pointer animate-fade-in bg-gray-100 dark:bg-dark-surface"
      style={{ animationDelay: `${(index % 12) * 50}ms` }}
      onClick={() => onClick(photo)}
    >
      <div className="aspect-[4/3] relative">
        <img
          ref={imgRef}
          src={imgSrc}
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
  )
})

import React from 'react'

export default function PhotoGrid({
  photos: propPhotos,
  loading: propLoading = false,
  dir = '',
  hasMore: propHasMore,
  onLoadMore: propOnLoadMore,
}: PhotoGridProps) {
  const [selectedPhoto, setSelectedPhoto] = useState<Photo | null>(null)
  const [visibleRange, setVisibleRange] = useState({ start: 0, end: 40 })
  const [realPhotos, setRealPhotos] = useState<Photo[]>([])
  const [columns, setColumns] = useState(getColumns())
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loadingMore, setLoadingMore] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const loadMoreRef = useRef<HTMLDivElement>(null)
  const scrollTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isSearchMode = !!propPhotos

  // 窗口大小变化 - 使用防抖
  useEffect(() => {
    let resizeTimeout: ReturnType<typeof setTimeout>
    const handleResize = () => {
      clearTimeout(resizeTimeout)
      resizeTimeout = setTimeout(() => {
        const newCols = getColumns()
        setColumns(newCols)
      }, 150)
    }
    window.addEventListener('resize', handleResize)
    return () => {
      window.removeEventListener('resize', handleResize)
      clearTimeout(resizeTimeout)
    }
  }, [])

  // 列数变化时重算可见范围
  useEffect(() => {
    if (containerRef.current) {
      const { scrollTop, clientHeight } = containerRef.current
      const rowHeight = ROW_HEIGHT_ESTIMATE + GAP
      const startRow = Math.max(0, Math.floor(scrollTop / rowHeight) - OVERSCAN_ROWS)
      const endRow = Math.ceil((scrollTop + clientHeight) / rowHeight) + OVERSCAN_ROWS
      const currentPhotos = propPhotos || realPhotos
      const start = startRow * columns
      const end = Math.min(currentPhotos.length, endRow * columns)
      setVisibleRange({ start, end })
    }
  }, [columns, propPhotos, realPhotos])

  const fetchPhotos = useCallback(async (targetPage: number, append: boolean) => {
    if (propPhotos) return
    setLoadingMore(true)
    try {
      const params = new URLSearchParams({
        page: String(targetPage),
        page_size: String(PAGE_SIZE),
      })
      if (dir) params.set('dir', dir)
      const res = await fetch(`/api/v1/photos?${params.toString()}`)
      const data = await res.json()
      const fetched = (data.photos || []).map((p: any) => ({
        id: p.ID || p.id,
        url: `/api/v1/photos/file?path=${encodeURIComponent(p.Path || p.path)}`,
        thumbnail: `/api/v1/photos/file?path=${encodeURIComponent(p.Path || p.path)}`,
        title: p.Name || p.name || '',
        date: p.DateTime || p.date_time || '',
        tags: p.Tags || p.tags || [],
        width: p.Width || p.width || 0,
        height: p.Height || p.height || 0,
        cameraMake: p.CameraMake || p.camera_make || '',
        cameraModel: p.CameraModel || p.camera_model || '',
      }))
      setRealPhotos(prev => append ? [...prev, ...fetched] : fetched)
      setTotal(data.total || 0)
      setPage(targetPage)
    } catch (err) {
      console.error('Failed to fetch photos:', err)
    } finally {
      setLoadingMore(false)
    }
  }, [propPhotos, dir])

  // 目录变化时重置并重新加载
  useEffect(() => {
    if (!propPhotos) {
      setRealPhotos([])
      setPage(1)
      setTotal(0)
      fetchPhotos(1, false)
    }
  }, [dir, propPhotos, fetchPhotos])

  // 初始加载
  useEffect(() => {
    if (!propPhotos && realPhotos.length === 0) {
      fetchPhotos(1, false)
    }
  }, [propPhotos, fetchPhotos, realPhotos.length])

  const photos = useMemo(() => {
    if (propPhotos) return propPhotos
    return realPhotos
  }, [propPhotos, realPhotos])

  const hasMore = isSearchMode
    ? (propHasMore ?? false)
    : (photos.length < total)

  const onLoadMore = isSearchMode
    ? propOnLoadMore
    : () => fetchPhotos(page + 1, true)

  // 无限滚动 - 使用 IntersectionObserver
  useEffect(() => {
    if (!loadMoreRef.current || !hasMore || !onLoadMore) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingMore) {
          onLoadMore()
        }
      },
      { root: containerRef.current, threshold: 0.1 }
    )
    observer.observe(loadMoreRef.current)
    return () => observer.disconnect()
  }, [hasMore, loadingMore, onLoadMore])

  // Virtual scroll - 使用防抖优化滚动计算
  const handleScroll = useCallback(() => {
    if (!containerRef.current) return

    if (scrollTimeoutRef.current) {
      clearTimeout(scrollTimeoutRef.current)
    }

    scrollTimeoutRef.current = setTimeout(() => {
      if (!containerRef.current) return
      const cols = getColumns()
      const rowHeight = ROW_HEIGHT_ESTIMATE + GAP
      const { scrollTop, clientHeight } = containerRef.current
      const startRow = Math.max(0, Math.floor(scrollTop / rowHeight) - OVERSCAN_ROWS)
      const endRow = Math.ceil((scrollTop + clientHeight) / rowHeight) + OVERSCAN_ROWS
      const start = startRow * cols
      const end = Math.min(photos.length, endRow * cols)
      setVisibleRange({ start, end })
    }, 16) // ~60fps
  }, [photos.length])

  useEffect(() => {
    const container = containerRef.current
    if (container) {
      container.addEventListener('scroll', handleScroll, { passive: true })
      return () => {
        container.removeEventListener('scroll', handleScroll)
        if (scrollTimeoutRef.current) {
          clearTimeout(scrollTimeoutRef.current)
        }
      }
    }
  }, [handleScroll])

  // Lightbox 键盘导航
  useEffect(() => {
    if (!selectedPhoto) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setSelectedPhoto(null)
        return
      }
      const idx = photos.findIndex(p => p.id === selectedPhoto.id)
      if (e.key === 'ArrowLeft' && idx > 0) setSelectedPhoto(photos[idx - 1])
      else if (e.key === 'ArrowRight' && idx < photos.length - 1) setSelectedPhoto(photos[idx + 1])
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedPhoto, photos])

  const topSpacerHeight = useMemo(() => {
    const rows = Math.floor(visibleRange.start / columns)
    return rows * (ROW_HEIGHT_ESTIMATE + GAP)
  }, [visibleRange.start, columns])

  const bottomSpacerHeight = useMemo(() => {
    const rows = Math.ceil((photos.length - visibleRange.end) / columns)
    return Math.max(0, rows * (ROW_HEIGHT_ESTIMATE + GAP))
  }, [photos.length, visibleRange.end, columns])

  const loading = propLoading || loadingMore

  if (loading && photos.length === 0) {
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
        <h3 className="text-lg font-semibold text-gray-900 dark:text-dark-text mb-2">
          {dir ? '该目录下没有照片' : '还没有照片'}
        </h3>
        <p className="text-sm text-gray-500 dark:text-dark-muted mb-6 max-w-sm">
          {dir ? '尝试选择其他目录或扫描新的照片目录' : '点击右上角的上传按钮，扫描你的照片目录开始使用'}
        </p>
      </div>
    )
  }

  const visiblePhotos = photos.slice(visibleRange.start, visibleRange.end)

  return (
    <>
      <div
        ref={containerRef}
        className="h-[calc(100vh-200px)] overflow-y-auto scrollbar-thin"
      >
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 p-4">
          {topSpacerHeight > 0 && (
            <div style={{ height: topSpacerHeight, gridColumn: `1 / -1` }} />
          )}
          {visiblePhotos.map((photo, index) => (
            <PhotoCard
              key={photo.id}
              photo={photo}
              index={visibleRange.start + index}
              onClick={setSelectedPhoto}
            />
          ))}
          {bottomSpacerHeight > 0 && (
            <div style={{ height: bottomSpacerHeight, gridColumn: `1 / -1` }} />
          )}

          {/* 无限滚动触发器 / 加载提示 */}
          {(hasMore || loadingMore || photos.length > 0) && (
            <div
              ref={loadMoreRef}
              className="col-span-full py-6 text-center"
            >
              {loadingMore ? (
                <span className="flex items-center justify-center gap-2 text-sm text-gray-500 dark:text-dark-muted">
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  加载更多照片中...
                </span>
              ) : hasMore ? (
                <span className="text-sm text-gray-400 dark:text-dark-muted">向下滚动加载更多</span>
              ) : photos.length > 0 ? (
                <span className="text-sm text-gray-400 dark:text-dark-muted">已加载全部 {photos.length} 张照片</span>
              ) : null}
            </div>
          )}
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
              {(selectedPhoto.width > 0 || selectedPhoto.height > 0 || selectedPhoto.cameraMake || selectedPhoto.cameraModel) && (
                <p className="text-white/60 text-xs mt-1">
                  {selectedPhoto.width > 0 && selectedPhoto.height > 0 && `${selectedPhoto.width} × ${selectedPhoto.height}  `}
                  {[selectedPhoto.cameraMake, selectedPhoto.cameraModel].filter(Boolean).join(' ')}
                </p>
              )}
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

            {photos.findIndex(p => p.id === selectedPhoto.id) > 0 && (
              <button
                onClick={() => {
                  const idx = photos.findIndex(p => p.id === selectedPhoto.id)
                  if (idx > 0) setSelectedPhoto(photos[idx - 1])
                }}
                className="absolute left-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white/20 hover:bg-white/30 text-white backdrop-blur-sm transition-colors"
              >
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
                </svg>
              </button>
            )}
            {photos.findIndex(p => p.id === selectedPhoto.id) < photos.length - 1 && (
              <button
                onClick={() => {
                  const idx = photos.findIndex(p => p.id === selectedPhoto.id)
                  if (idx < photos.length - 1) setSelectedPhoto(photos[idx + 1])
                }}
                className="absolute right-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white/20 hover:bg-white/30 text-white backdrop-blur-sm transition-colors"
              >
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                </svg>
              </button>
            )}
          </div>
        </div>
      )}
    </>
  )
}
