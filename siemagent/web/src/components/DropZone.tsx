import { useRef, useState, useCallback } from 'react'
import { Upload } from 'lucide-react'
import { classifyLog } from '../lib/api'
import type { ClassifiedEvent } from '../lib/api'

interface Props {
  onResults: (events: ClassifiedEvent[]) => void
}

export function DropZone({ onResults }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null)
  const [toast, setToast] = useState<string | null>(null)

  const process = useCallback(async (file: File) => {
    const text = await file.text()
    const lines = text
      .split('\n')
      .map((l) => l.trim())
      .filter((l) => l && !l.startsWith('#'))

    if (!lines.length) return

    const total = lines.length
    setProgress({ done: 0, total })

    const BATCH = 5
    const results: ClassifiedEvent[] = []
    const counts = { P1: 0, P2: 0, high: 0, other: 0 }

    for (let i = 0; i < lines.length; i += BATCH) {
      const batch = lines.slice(i, i + BATCH)
      const settled = await Promise.allSettled(batch.map((l) => classifyLog(l)))
      for (const res of settled) {
        if (res.status === 'fulfilled') {
          results.push(res.value)
          const sev = res.value.severity
          if (sev === 'P1') counts.P1++
          else if (sev === 'P2') counts.P2++
          else if (sev === 'P3') counts.high++
          else counts.other++
        }
      }
      setProgress({ done: Math.min(i + BATCH, total), total })
    }

    onResults(results)
    setProgress(null)
    const msg = `${results.length} events classified — ${counts.P1} Critical, ${counts.P2} High, ${counts.high + counts.other} others`
    setToast(msg)
    setTimeout(() => setToast(null), 6000)
  }, [onResults])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragging(false)
    const file = e.dataTransfer.files[0]
    if (file) process(file)
  }, [process])

  const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) process(file)
    e.target.value = ''
  }, [process])

  return (
    <div className="relative">
      <div
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        className={`flex items-center gap-2 cursor-pointer px-3 py-2 rounded border border-dashed transition-colors ${
          dragging
            ? 'border-blue-500 bg-blue-900/20'
            : 'border-gray-700 hover:border-gray-500'
        }`}
      >
        <Upload size={16} className="text-gray-500" />
        <span className="text-sm text-gray-400">
          {progress
            ? `Classifying ${progress.done} / ${progress.total} events…`
            : 'Upload .log or .txt'}
        </span>
        {progress && (
          <div className="ml-2 h-1.5 w-24 bg-gray-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all"
              style={{ width: `${(progress.done / progress.total) * 100}%` }}
            />
          </div>
        )}
      </div>

      <input
        ref={inputRef}
        type="file"
        accept=".log,.txt"
        className="hidden"
        onChange={handleChange}
      />

      {toast && (
        <div className="absolute top-full mt-1 left-0 z-50 bg-gray-800 border border-gray-700 text-sm text-gray-200 px-3 py-2 rounded shadow-lg whitespace-nowrap">
          {toast}
        </div>
      )}
    </div>
  )
}
