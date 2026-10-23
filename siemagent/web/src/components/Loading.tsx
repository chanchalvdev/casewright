import { BrandMark } from './BrandMark'

// Spinner is a ring with a moving arc. It inherits colour from the text, so
// it works on accent buttons and on plain surfaces alike.
export function Spinner({ size = 16, className = '' }: { size?: number; className?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true" className={`shrink-0 animate-spin ${className}`}>
      <circle cx="12" cy="12" r="9" stroke="currentColor" strokeOpacity="0.2" strokeWidth="3" />
      <path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
    </svg>
  )
}

// LoadingState announces a load in progress, for panels and page bodies.
export function LoadingState({ label = 'Loading…', compact = false, className = '' }: { label?: string; compact?: boolean; className?: string }) {
  return (
    <div
      role="status"
      aria-live="polite"
      className={`flex items-center gap-2.5 text-fg-subtle ${compact ? 'text-xs' : 'justify-center py-10 text-sm'} ${className}`}
    >
      <Spinner size={compact ? 13 : 18} className="text-accent-text" />
      <span>{label}</span>
    </div>
  )
}

// BootSplash covers the screen while the app works out who is signed in.
export function BootSplash({ label = 'Checking your session…' }: { label?: string }) {
  return (
    <main role="status" aria-live="polite" className="flex h-screen flex-col items-center justify-center gap-5 bg-canvas">
      <div className="relative flex h-20 w-20 items-center justify-center">
        <span className="boot-halo absolute inset-0 rounded-[22px] bg-accent/20" aria-hidden="true" />
        <svg viewBox="0 0 80 80" className="absolute inset-0 animate-spin text-accent-text [animation-duration:1.4s]" aria-hidden="true">
          <circle cx="40" cy="40" r="37" fill="none" stroke="currentColor" strokeOpacity="0.15" strokeWidth="2" />
          <path d="M77 40A37 37 0 0 0 40 3" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" />
        </svg>
        <BrandMark size={44} />
      </div>
      <div className="text-center">
        <p className="text-sm font-semibold tracking-tight text-fg">Casewright</p>
        <p className="mt-0.5 text-xs text-fg-subtle">{label}</p>
      </div>
    </main>
  )
}
