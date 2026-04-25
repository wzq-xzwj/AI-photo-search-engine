import { useState, useRef, useMemo } from 'react'
import type { FaceInfo } from '../services/faces'

interface FaceOverlayProps {
  faces: FaceInfo[]
  imageWidth: number
  imageHeight: number
  displayWidth: number
  displayHeight: number
  onLabelRequest?: (face: FaceInfo) => void
  onNavigateToPerson?: (personId: string) => void
}

export default function FaceOverlay({
  faces,
  imageWidth,
  imageHeight,
  displayWidth,
  displayHeight,
  onLabelRequest,
  onNavigateToPerson,
}: FaceOverlayProps) {
  const [hoveredFaceIndex, setHoveredFaceIndex] = useState<number | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const scaleX = useMemo(
    () => displayWidth / (imageWidth || 1),
    [displayWidth, imageWidth]
  )
  const scaleY = useMemo(
    () => displayHeight / (imageHeight || 1),
    [displayHeight, imageHeight]
  )

  if (!faces.length || !imageWidth || !imageHeight) return null

  return (
    <div
      ref={containerRef}
      className="absolute inset-0 pointer-events-none"
    >
      {faces.map((face) => {
        const { location } = face
        const left = location.left * scaleX
        const top = location.top * scaleY
        const width = (location.right - location.left) * scaleX
        const height = (location.bottom - location.top) * scaleY

        const isHovered = hoveredFaceIndex === face.face_index
        const isLabeled = !!face.person_id

        return (
          <div
            key={face.face_index}
            className="absolute pointer-events-auto"
            style={{ left, top, width, height }}
            onMouseEnter={() => setHoveredFaceIndex(face.face_index)}
            onMouseLeave={() => setHoveredFaceIndex(null)}
          >
            {/* Face bounding box */}
            <div
              className={`absolute inset-0 rounded-sm border-2 transition-all duration-200 ${
                isLabeled
                  ? isHovered
                    ? 'border-green-400 shadow-lg shadow-green-400/30'
                    : 'border-green-400/60'
                  : isHovered
                    ? 'border-amber-400 shadow-lg shadow-amber-400/30'
                    : 'border-amber-400/50'
              }`}
            />

            {/* Hover tooltip */}
            {isHovered && (
              <div
                className="absolute -top-8 left-1/2 -translate-x-1/2 flex items-center gap-1 px-2 py-1 rounded-md bg-black/80 text-white text-xs whitespace-nowrap shadow-lg"
              >
                {isLabeled ? (
                  <>
                    <span className="font-medium">{face.person_name}</span>
                    {onNavigateToPerson && face.person_id && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          onNavigateToPerson(face.person_id!)
                        }}
                        className="ml-1 px-1.5 py-0.5 rounded bg-white/20 hover:bg-white/30 text-xs transition-colors"
                        title="查看此人物"
                      >
                        →
                      </button>
                    )}
                  </>
                ) : (
                  <>
                    <span>未标注 #{face.face_index + 1}</span>
                    {onLabelRequest && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          onLabelRequest(face)
                        }}
                        className="ml-1 px-1.5 py-0.5 rounded bg-amber-500/80 hover:bg-amber-500 text-xs transition-colors"
                      >
                        标注
                      </button>
                    )}
                  </>
                )}
              </div>
            )}

            {/* Confidence badge */}
            {isHovered && (
              <div className="absolute -bottom-5 left-1/2 -translate-x-1/2 px-1.5 py-0.5 rounded bg-black/60 text-white text-[10px]">
                {Math.round(face.confidence * 100)}%
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
