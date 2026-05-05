import { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import {
  getPersons,
  searchPhotosByPerson,
  getFaceThumbnailUrlByID,
  type Person,
} from '../services/faces'
import { useDebounce } from '../hooks/useDebounce'
import PhotoGrid from '../components/PhotoGrid'

// 聚类后的人物类型
interface ClusteredPerson {
  person_id: string
  face_count: number
  photo_count: number
  sample_face: {
    id: number
    photo_id: string
    face_index: number
    thumbnail_url: string
  } | null
  name: string
}

export default function Persons() {
  const [persons, setPersons] = useState<ClusteredPerson[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const debouncedQuery = useDebounce(searchQuery, 300)

  // Person detail view
  const [selectedPerson, setSelectedPerson] = useState<ClusteredPerson | null>(null)
  const [personPhotos, setPersonPhotos] = useState<any[]>([])
  const [personPhotoTotal, setPersonPhotoTotal] = useState(0)
  const [loadingPhotos, setLoadingPhotos] = useState(false)

  // Label modal
  const [labelingPerson, setLabelingPerson] = useState<ClusteredPerson | null>(null)
  const [labelName, setLabelName] = useState('')
  const [labeling, setLabeling] = useState(false)

  const loadPersons = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      // 从聚类结果获取人物列表
      const res = await fetch('/api/v1/faces/clusters')
      if (!res.ok) throw new Error('加载人物列表失败')
      const data = await res.json()
      const clusters: ClusteredPerson[] = data.data?.clusters || []
      // 过滤掉没有人脸样本的人物（避免显示非人脸照片）
      const validClusters = clusters.filter((p: ClusteredPerson) => p.sample_face !== null && p.face_count > 0)
      setPersons(validClusters)
      setTotal(validClusters.length)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载人物列表失败')
      // 降级：使用旧 API
      try {
        const result = await getPersons(200)
        const fallback = result.persons.map((p: Person) => ({
          person_id: p.id,
          face_count: p.face_count,
          photo_count: p.photo_count,
          sample_face: null,
          name: p.name,
        }))
        setPersons(fallback)
        setTotal(result.total)
      } catch {
        // ignore
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadPersons()
  }, [loadPersons])

  const loadPersonPhotos = useCallback(async (personId: string) => {
    setLoadingPhotos(true)
    try {
      const result = await searchPhotosByPerson(personId, 50)
      setPersonPhotos(result.photos)
      setPersonPhotoTotal(result.total)
    } catch {
      setPersonPhotos([])
    } finally {
      setLoadingPhotos(false)
    }
  }, [])

  const handleSelectPerson = (person: ClusteredPerson) => {
    setSelectedPerson(person)
    loadPersonPhotos(person.person_id)
  }

  const handleBack = () => {
    setSelectedPerson(null)
    setPersonPhotos([])
    setPersonPhotoTotal(0)
  }

  const handleLabel = async () => {
    if (!labelingPerson || !labelName.trim()) return
    setLabeling(true)
    try {
      // 标注该人物的所有样本人脸
      const res = await fetch(`/api/v1/faces/label-cluster`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          person_id: labelingPerson.person_id,
          person_name: labelName.trim(),
        }),
      })
      if (!res.ok) throw new Error('标注失败')
      
      // 更新本地状态
      setPersons(prev => prev.map(p => 
        p.person_id === labelingPerson.person_id 
          ? { ...p, name: labelName.trim() } 
          : p
      ))
      setLabelingPerson(null)
      setLabelName('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '标注失败')
    } finally {
      setLabeling(false)
    }
  }

  const filteredPersons = debouncedQuery
    ? persons.filter((p) =>
        p.name.toLowerCase().includes(debouncedQuery.toLowerCase()) ||
        (p.name === '未知人物' && '未知'.includes(debouncedQuery))
      )
    : persons

  // Person detail view
  if (selectedPerson) {
    return (
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Back button and header */}
        <div className="flex items-center gap-4 mb-6">
          <button
            onClick={handleBack}
            className="p-2 rounded-xl bg-gray-100 dark:bg-dark-surface hover:bg-gray-200 dark:hover:bg-dark-card transition-colors"
          >
            <svg className="w-5 h-5 text-gray-600 dark:text-dark-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <div className="flex-1">
            <h1 className="text-2xl font-display font-bold text-gray-900 dark:text-dark-text">
              {selectedPerson.name}
            </h1>
            <p className="text-sm text-gray-500 dark:text-dark-muted">
              {selectedPerson.face_count} 张人脸 · {personPhotoTotal} 张照片
            </p>
          </div>
          <button
            onClick={() => {
              setLabelingPerson(selectedPerson)
              setLabelName(selectedPerson.name === '未知人物' ? '' : selectedPerson.name)
            }}
            className="px-4 py-2 rounded-xl bg-primary-500 text-white text-sm font-medium hover:bg-primary-600 transition-colors"
          >
            标注姓名
          </button>
        </div>

        {/* Photo Grid */}
        {loadingPhotos ? (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="aspect-[4/3] rounded-xl bg-gray-200 dark:bg-dark-surface animate-pulse" />
            ))}
          </div>
        ) : personPhotos.length === 0 ? (
          <div className="text-center py-20">
            <svg className="w-16 h-16 text-gray-300 dark:text-dark-border mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
            </svg>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-dark-text mb-2">暂未关联照片</h3>
            <p className="text-sm text-gray-500 dark:text-dark-muted">标注人脸后，照片会自动关联到对应人物</p>
          </div>
        ) : (
          <PhotoGrid photos={personPhotos} />
        )}
      </main>
    )
  }

  // Person list
  return (
    <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-2">
          <div>
            <h1 className="text-3xl font-display font-bold text-gray-900 dark:text-dark-text">
              人物
            </h1>
            <p className="text-sm text-gray-500 dark:text-dark-muted mt-1">
              {total > 0 ? `共 ${total} 位人物` : '扫描照片后，人物将自动显示在这里'}
            </p>
          </div>
          {total > 0 && (
            <button
              onClick={loadPersons}
              disabled={loading}
              className="px-3 py-2 text-sm rounded-xl bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-card transition-colors flex items-center gap-1"
            >
              <svg className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              刷新
            </button>
          )}
        </div>

        {/* Search */}
        {total > 0 && (
          <div className="relative max-w-md">
            <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="搜索人物..."
              className="w-full pl-10 pr-4 py-2.5 text-sm rounded-xl border border-gray-200 dark:border-dark-border bg-white dark:bg-dark-card text-gray-900 dark:text-dark-text focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
        )}
      </div>

      {error && (
        <div className="mb-6 p-4 rounded-xl bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 text-sm">
          {error}
        </div>
      )}

      {loading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {Array.from({ length: 10 }).map((_, i) => (
            <div key={i} className="animate-pulse">
              <div className="aspect-square rounded-2xl bg-gray-200 dark:bg-dark-surface mb-3" />
              <div className="h-4 w-2/3 rounded bg-gray-200 dark:bg-dark-surface mx-auto" />
            </div>
          ))}
        </div>
      ) : filteredPersons.length === 0 ? (
        <div className="text-center py-20">
          <div className="w-20 h-20 rounded-2xl bg-gray-100 dark:bg-dark-surface flex items-center justify-center mx-auto mb-6">
            <svg className="w-10 h-10 text-gray-400 dark:text-dark-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
          </div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-dark-text mb-2">
            {searchQuery ? '未找到匹配的人物' : '暂无人物'}
          </h3>
          <p className="text-sm text-gray-500 dark:text-dark-muted max-w-sm mx-auto">
            {searchQuery
              ? '尝试其他搜索词'
              : '扫描照片后，人脸会自动聚类并显示在这里'}
          </p>
          {!searchQuery && (
            <Link
              to="/"
              className="inline-block mt-6 px-6 py-3 rounded-xl bg-primary-500 text-white text-sm font-medium hover:bg-primary-600 transition-colors"
            >
              返回首页
            </Link>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {filteredPersons.map((person) => (
            <div
              key={person.person_id}
              className="group relative p-3 rounded-2xl bg-white dark:bg-dark-card border border-gray-100 dark:border-dark-border hover:border-primary-300 dark:hover:border-primary-600 hover:shadow-md transition-all duration-200"
            >
              {/* Face thumbnail */}
              <button
                onClick={() => handleSelectPerson(person)}
                className="w-full aspect-square rounded-xl overflow-hidden mb-3 bg-gray-100 dark:bg-dark-surface"
              >
                {person.sample_face ? (
                  <img
                    src={getFaceThumbnailUrlByID(person.sample_face.id)}
                    alt={person.name}
                    className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    loading="lazy"
                  />
                ) : (
                  <div className="w-full h-full flex items-center justify-center">
                    <svg className="w-12 h-12 text-gray-300 dark:text-dark-border" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                  </div>
                )}
              </button>

              {/* Name and actions */}
              <div className="text-center">
                <h3 className="text-sm font-semibold text-gray-900 dark:text-dark-text truncate">
                  {person.name}
                </h3>
                <p className="text-xs text-gray-500 dark:text-dark-muted mt-0.5">
                  {person.face_count} 张人脸 · {person.photo_count} 张照片
                </p>
              </div>

              {/* Label button */}
              <button
                onClick={() => {
                  setLabelingPerson(person)
                  setLabelName(person.name === '未知人物' ? '' : person.name)
                }}
                className="absolute top-2 right-2 p-1.5 rounded-lg bg-white/90 dark:bg-dark-card/90 backdrop-blur-sm shadow-sm opacity-0 group-hover:opacity-100 transition-opacity"
                title="标注姓名"
              >
                <svg className="w-4 h-4 text-gray-600 dark:text-dark-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
                </svg>
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Label Modal */}
      {labelingPerson && (
        <div className="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-4 animate-fade-in">
          <div
            className="bg-white dark:bg-dark-card rounded-2xl shadow-2xl border border-gray-100 dark:border-dark-border w-full max-w-md animate-slide-up"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-6">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-dark-text mb-4">
                标注人物姓名
              </h3>
              
              {/* Preview */}
              {labelingPerson.sample_face && (
                <div className="flex justify-center mb-4">
                  <img
                    src={getFaceThumbnailUrlByID(labelingPerson.sample_face.id)}
                    alt="Preview"
                    className="w-24 h-24 rounded-xl object-cover"
                  />
                </div>
              )}
              
              <input
                type="text"
                value={labelName}
                onChange={(e) => setLabelName(e.target.value)}
                placeholder="输入姓名..."
                className="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-dark-border bg-white dark:bg-dark-surface text-gray-900 dark:text-dark-text focus:outline-none focus:ring-2 focus:ring-primary-500 mb-4"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleLabel()
                }}
              />
              <div className="flex gap-3">
                <button
                  onClick={() => {
                    setLabelingPerson(null)
                    setLabelName('')
                  }}
                  className="flex-1 px-4 py-2.5 rounded-xl bg-gray-100 dark:bg-dark-surface text-gray-700 dark:text-dark-muted font-medium hover:bg-gray-200 dark:hover:bg-dark-card transition-colors"
                >
                  取消
                </button>
                <button
                  onClick={handleLabel}
                  disabled={!labelName.trim() || labeling}
                  className="flex-1 px-4 py-2.5 rounded-xl bg-primary-500 text-white font-medium hover:bg-primary-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {labeling ? '保存中...' : '保存'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </main>
  )
}
