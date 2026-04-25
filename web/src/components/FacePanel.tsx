import { useState, useEffect, useCallback } from 'react'
import {
  detectFaces,
  labelFace,
  getFaceThumbnailUrl,
  getPersons,
  type FaceInfo,
  type Person,
} from '../services/faces'

interface FacePanelProps {
  photoId: string
  onFacesDetected?: (faces: FaceInfo[]) => void
}

export default function FacePanel({ photoId, onFacesDetected }: FacePanelProps) {
  const [faces, setFaces] = useState<FaceInfo[]>([])
  const [persons, setPersons] = useState<Person[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [detecting, setDetecting] = useState(false)
  const [labelingFaceId, setLabelingFaceId] = useState<number | null>(null)
  const [labelName, setLabelName] = useState('')
  const [selectedPersonId, setSelectedPersonId] = useState('')
  const [showNewPersonInput, setShowNewPersonInput] = useState(false)

  const loadFaces = useCallback(async () => {
    if (!photoId) return
    setLoading(true)
    setError('')
    try {
      const result = await detectFaces(photoId)
      setFaces(result.faces || [])
      onFacesDetected?.(result.faces || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载人脸数据失败')
    } finally {
      setLoading(false)
    }
  }, [photoId, onFacesDetected])

  const loadPersons = useCallback(async () => {
    try {
      const result = await getPersons(100)
      setPersons(result.persons)
    } catch {
      // 人物列表加载失败不影响主功能
    }
  }, [])

  useEffect(() => {
    loadFaces()
    loadPersons()
  }, [loadFaces, loadPersons])

  const handleDetect = async () => {
    setDetecting(true)
    setError('')
    try {
      const result = await detectFaces(photoId)
      setFaces(result.faces || [])
      onFacesDetected?.(result.faces || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '人脸检测失败')
    } finally {
      setDetecting(false)
    }
  }

  const handleLabel = async (faceId: number) => {
    if (!labelName.trim() && !selectedPersonId) return

    try {
      const result = await labelFace(
        faceId,
        selectedPersonId ? undefined : labelName.trim(),
        selectedPersonId || undefined
      )

      // 更新本地状态
      setFaces((prev) =>
        prev.map((f) =>
          f.id === faceId
            ? {
                ...f,
                person_id: result.person_id,
                person_name:
                  result.person_name ||
                  labelName.trim() ||
                  persons.find((p) => p.id === selectedPersonId)?.name,
              }
            : f
        )
      )

      // 刷新人物列表
      loadPersons()

      // 重置表单
      setLabelingFaceId(null)
      setLabelName('')
      setSelectedPersonId('')
      setShowNewPersonInput(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : '标注失败')
    }
  }

  const unlabeledFaces = faces.filter((f) => !f.person_id)
  const labeledFaces = faces.filter((f) => f.person_id)

  if (loading) {
    return (
      <div className="p-4">
        <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-dark-muted">
          <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          加载人脸数据...
        </div>
      </div>
    )
  }

  return (
    <div className="bg-white dark:bg-dark-card rounded-xl border border-gray-100 dark:border-dark-border overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100 dark:border-dark-border">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-dark-text flex items-center gap-2">
          <svg className="w-4 h-4 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
          </svg>
          人脸识别
          {faces.length > 0 && (
            <span className="text-xs text-gray-400 dark:text-dark-muted">({faces.length})</span>
          )}
        </h3>
        <button
          onClick={handleDetect}
          disabled={detecting}
          className="px-3 py-1.5 text-xs font-medium rounded-lg bg-primary-500 text-white hover:bg-primary-600 disabled:opacity-50 transition-colors flex items-center gap-1"
        >
          {detecting ? (
            <>
              <svg className="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              检测中...
            </>
          ) : (
            '重新检测'
          )}
        </button>
      </div>

      {error && (
        <div className="px-4 py-2 bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 text-xs">
          {error}
        </div>
      )}

      {/* Content */}
      <div className="p-4 space-y-4">
        {faces.length === 0 && !loading && (
          <div className="text-center py-6">
            <svg className="w-10 h-10 text-gray-300 dark:text-dark-border mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
            <p className="text-sm text-gray-400 dark:text-dark-muted">暂无人脸数据</p>
            <p className="text-xs text-gray-400 dark:text-dark-muted mt-1">点击"重新检测"开始检测人脸</p>
          </div>
        )}

        {/* Unlabeled faces */}
        {unlabeledFaces.length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-gray-500 dark:text-dark-muted mb-2 uppercase tracking-wide">
              待标注 ({unlabeledFaces.length})
            </h4>
            <div className="grid grid-cols-3 sm:grid-cols-4 gap-2">
              {unlabeledFaces.map((face) => (
                <div key={face.face_index} className="relative group">
                  <img
                    src={
                      face.thumbnail_url ||
                      getFaceThumbnailUrl(photoId, face.face_index)
                    }
                    alt={`Face ${face.face_index + 1}`}
                    className="w-full aspect-square object-cover rounded-lg border-2 border-dashed border-amber-300 dark:border-amber-600"
                  />
                  <button
                    onClick={() => {
                      setLabelingFaceId(labelingFaceId === face.id ? null : face.id)
                      setShowNewPersonInput(false)
                    }}
                    className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 group-hover:opacity-100 rounded-lg transition-opacity"
                  >
                    <span className="text-xs text-white font-medium bg-primary-500 px-2 py-1 rounded-md">
                      标注
                    </span>
                  </button>

                  {/* Label dialog */}
                  {labelingFaceId === face.id && (
                    <div className="absolute top-full left-0 right-0 mt-2 bg-white dark:bg-dark-card rounded-lg shadow-xl border border-gray-200 dark:border-dark-border p-3 z-50 min-w-[200px]">
                      {!showNewPersonInput ? (
                        <>
                          {/* Existing persons dropdown */}
                          {persons.length > 0 && (
                            <select
                              value={selectedPersonId}
                              onChange={(e) => setSelectedPersonId(e.target.value)}
                              className="w-full text-sm px-3 py-2 rounded-lg border border-gray-200 dark:border-dark-border bg-white dark:bg-dark-surface text-gray-900 dark:text-dark-text mb-2 focus:outline-none focus:ring-2 focus:ring-primary-500"
                            >
                              <option value="">选择已有的人物...</option>
                              {persons.map((p) => (
                                <option key={p.id} value={p.id}>
                                  {p.name} ({p.face_count}张)
                                </option>
                              ))}
                            </select>
                          )}

                          <button
                            onClick={() => setShowNewPersonInput(true)}
                            className="w-full text-xs text-primary-500 hover:text-primary-600 py-1"
                          >
                            + 创建新人物
                          </button>

                          <div className="flex gap-2 mt-2">
                            <button
                              onClick={() => handleLabel(face.id)}
                              disabled={!selectedPersonId}
                              className="flex-1 px-3 py-1.5 text-xs font-medium rounded-lg bg-primary-500 text-white hover:bg-primary-600 disabled:opacity-50 transition-colors"
                            >
                              确认
                            </button>
                            <button
                              onClick={() => {
                                setLabelingFaceId(null)
                                setSelectedPersonId('')
                              }}
                              className="px-3 py-1.5 text-xs rounded-lg bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-border"
                            >
                              取消
                            </button>
                          </div>
                        </>
                      ) : (
                        <>
                          <input
                            type="text"
                            value={labelName}
                            onChange={(e) => setLabelName(e.target.value)}
                            placeholder="输入人物姓名"
                            className="w-full text-sm px-3 py-2 rounded-lg border border-gray-200 dark:border-dark-border bg-white dark:bg-dark-surface text-gray-900 dark:text-dark-text mb-2 focus:outline-none focus:ring-2 focus:ring-primary-500"
                            autoFocus
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') handleLabel(face.id)
                            }}
                          />
                          <div className="flex gap-2">
                            <button
                              onClick={() => handleLabel(face.id)}
                              disabled={!labelName.trim()}
                              className="flex-1 px-3 py-1.5 text-xs font-medium rounded-lg bg-primary-500 text-white hover:bg-primary-600 disabled:opacity-50 transition-colors"
                            >
                              创建并标注
                            </button>
                            <button
                              onClick={() => {
                                setShowNewPersonInput(false)
                                setLabelName('')
                              }}
                              className="px-3 py-1.5 text-xs rounded-lg bg-gray-100 dark:bg-dark-surface text-gray-600 dark:text-dark-muted hover:bg-gray-200 dark:hover:bg-dark-border"
                            >
                              返回
                            </button>
                          </div>
                        </>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Labeled faces */}
        {labeledFaces.length > 0 && (
          <div>
            <h4 className="text-xs font-medium text-gray-500 dark:text-dark-muted mb-2 uppercase tracking-wide">
              已标注 ({labeledFaces.length})
            </h4>
            <div className="grid grid-cols-3 sm:grid-cols-4 gap-2">
              {labeledFaces.map((face) => (
                <div key={face.face_index} className="relative group">
                  <img
                    src={
                      face.thumbnail_url ||
                      getFaceThumbnailUrl(photoId, face.face_index)
                    }
                    alt={face.person_name || `Face ${face.face_index + 1}`}
                    className="w-full aspect-square object-cover rounded-lg border-2 border-green-400 dark:border-green-600"
                  />
                  <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/60 to-transparent rounded-b-md px-2 py-1">
                    <span className="text-xs text-white font-medium truncate block">
                      {face.person_name}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
