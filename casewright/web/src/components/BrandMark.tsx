import { useId } from 'react'

// BrandMark is the Casewright logo: an open "C" (the case) whose check stroke
// leaves through the opening (the case closed). Same geometry as
// public/favicon.svg; keep the two in step.
export function BrandMark({ size = 32, className = '', title }: { size?: number; className?: string; title?: string }) {
  const id = useId()
  const fill = `${id}-fill`
  const shine = `${id}-shine`
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      className={`shrink-0 ${className}`}
      role={title ? 'img' : undefined}
      aria-label={title}
      aria-hidden={title ? undefined : true}
    >
      <defs>
        <linearGradient id={fill} x1="4" y1="2" x2="28" y2="30" gradientUnits="userSpaceOnUse">
          <stop stopColor="#3B82F6" />
          <stop offset="1" stopColor="#4338CA" />
        </linearGradient>
        <linearGradient id={shine} x1="16" y1="0" x2="16" y2="16" gradientUnits="userSpaceOnUse">
          <stop stopColor="#fff" stopOpacity="0.28" />
          <stop offset="1" stopColor="#fff" stopOpacity="0" />
        </linearGradient>
      </defs>
      <rect width="32" height="32" rx="8" fill={`url(#${fill})`} />
      <rect x="0.5" y="0.5" width="31" height="31" rx="7.5" stroke={`url(#${shine})`} />
      <path d="M19.38 8.75A8 8 0 1 0 21.66 21.66" stroke="#fff" strokeWidth="2.75" strokeLinecap="round" />
      <path d="M11.5 16.2l3.3 3.3 9.7-9.7" stroke="#fff" strokeWidth="2.75" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

// Wordmark pairs the mark with the product name, as used in app chrome.
export function Wordmark({ tagline, size = 32, inverted = false }: { tagline?: string; size?: number; inverted?: boolean }) {
  return (
    <div className="flex items-center gap-3">
      <BrandMark size={size} />
      <div className="leading-tight">
        <p className={`font-semibold tracking-tight ${inverted ? 'text-white' : 'text-fg'} ${size >= 36 ? 'text-lg' : 'text-sm'}`}>Casewright</p>
        {tagline && <p className={`text-[11px] ${inverted ? 'text-white/60' : 'text-fg-subtle'}`}>{tagline}</p>}
      </div>
    </div>
  )
}
