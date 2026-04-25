const API_BASE = '/api/v1'

export interface FaceLocation {
  top: number
  right: number
  bottom: number
  left: number
}

export interface FaceInfo {
  id: number
  photo_id: string
  face_index: number
  person_id?: string
  person_name?: string
  location: FaceLocation
  confidence: number
  thumbnail_url?: string
}

export interface FaceDetectResponse {
  photo_id: string
  face_count: number
  faces: FaceInfo[]
}

export interface Person {
  id: string
  name: string
  avatar?: string
  face_count: number
  photo_count: number
  created_at: string
}

export interface PersonListResponse {
  total: number
  persons: Person[]
}

export interface PersonPhotoSearchResponse {
  total: number
  photos: Array<{
    id: string
    url: string
    thumbnail: string
    title: string
    date: string
    tags: string[]
    width: number
    height: number
  }>
}

/** 检测照片中的人脸 */
export async function detectFaces(photoId: string): Promise<FaceDetectResponse> {
  const res = await fetch(`${API_BASE}/faces/detect`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ photo_id: photoId }),
  })
  if (!res.ok) {
    throw new Error(`Face detection failed: ${res.status}`)
  }
  const data = await res.json()
  return data.data || data
}

/** 获取人脸缩略图 URL */
export function getFaceThumbnailUrl(photoId: string, faceIndex: number): string {
  return `${API_BASE}/faces/thumbnail?photo_id=${encodeURIComponent(photoId)}&index=${faceIndex}`
}

/** 标注人脸 */
export async function labelFace(
  faceId: number,
  personName?: string,
  personId?: string
): Promise<{ face_id: number; person_id: string; person_name?: string }> {
  const body: Record<string, string> = {}
  if (personId) body.person_id = personId
  else if (personName) body.person_name = personName
  else throw new Error('person_name or person_id required')

  const res = await fetch(`${API_BASE}/faces/${faceId}/label`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Label face failed: ${res.status}`)
  const data = await res.json()
  return data.data || data
}

/** 获取人物列表 */
export async function getPersons(limit = 50, offset = 0): Promise<PersonListResponse> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  const res = await fetch(`${API_BASE}/persons?${params}`)
  if (!res.ok) throw new Error(`Get persons failed: ${res.status}`)
  const data = await res.json()
  return {
    total: data.data?.total || data.total || 0,
    persons: data.data?.persons || data.persons || [],
  }
}

/** 按人物搜索照片 */
export async function searchPhotosByPerson(
  personId: string,
  limit = 20,
  offset = 0
): Promise<PersonPhotoSearchResponse> {
  const params = new URLSearchParams({
    person_id: personId,
    limit: String(limit),
    offset: String(offset),
  })
  const res = await fetch(`${API_BASE}/search/person?${params}`)
  if (!res.ok) throw new Error(`Search by person failed: ${res.status}`)
  const data = await res.json()
  const rawPhotos = data.data?.photos || data.photos || []
  const photos = rawPhotos.map((p: any) => ({
    id: p.ID || p.id,
    url: `/api/v1/photos/file?path=${encodeURIComponent(p.Path || p.path)}`,
    thumbnail: `/api/v1/photos/file?path=${encodeURIComponent(p.Path || p.path)}`,
    title: p.Name || p.name || '',
    date: p.DateTime || p.date_time || '',
    tags: p.Tags || p.tags || [],
    width: p.Width || p.width || 0,
    height: p.Height || p.height || 0,
  }))
  return {
    total: data.data?.total || data.total || 0,
    photos,
  }
}
