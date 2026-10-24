import { useState } from 'react'
import type { KeyboardEvent, ReactNode } from 'react'
import { ArrowRight, BellRing, BrainCircuit, CircleAlert, Eye, EyeOff, FileCheck, FolderSearch, LockKeyhole, ScanSearch, UserCheck, UserRound } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { apiError, login } from '../lib/api'
import { Wordmark } from '../components/BrandMark'
import { Spinner } from '../components/Loading'
import { ThemeToggle } from '../components/ThemeToggle'

const FLOW: { Icon: LucideIcon; label: string }[] = [
  { Icon: BellRing, label: 'Alert' },
  { Icon: FolderSearch, label: 'Case' },
  { Icon: FileCheck, label: 'Rule' },
]

const FEATURES: { Icon: LucideIcon; title: string; detail: string }[] = [
  { Icon: ScanSearch, title: 'Rules first', detail: 'Sigma rules match before the LLM is asked, so known attacks cost nothing.' },
  { Icon: BrainCircuit, title: 'One investigation per incident', detail: 'Related alerts are correlated into a single story with ATT&CK context.' },
  { Icon: UserCheck, title: 'Humans approve the response', detail: 'Containment waits for a sign-off, and every decision is audited.' },
]

// LoginPage is the sign-in screen: the product story on the left (large
// screens only) and the form on the right.
export function LoginPage({ onSuccess }: { onSuccess: () => void }) {
  return (
    <main className="grid min-h-screen bg-canvas lg:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)]">
      <BrandPanel />
      <section className="relative flex flex-col px-5 py-6 sm:px-10">
        <div className="flex items-center justify-between lg:justify-end">
          <div className="lg:hidden"><Wordmark size={30} /></div>
          <ThemeToggle compact />
        </div>
        <div className="flex flex-1 items-center justify-center py-10">
          <LoginForm onSuccess={onSuccess} />
        </div>
        <p className="text-center text-[11px] text-fg-subtle">Casewright · open-source, local-first AI security analyst</p>
      </section>
    </main>
  )
}

function BrandPanel() {
  return (
    <aside className="brand-panel relative hidden overflow-hidden lg:flex lg:flex-col lg:justify-between lg:gap-10 lg:p-12 xl:p-16">
      <div className="brand-grid pointer-events-none absolute inset-0" aria-hidden="true" />
      <div className="brand-glow pointer-events-none absolute -left-32 -top-32 h-[28rem] w-[28rem] rounded-full" aria-hidden="true" />

      <div className="relative"><Wordmark size={38} tagline="AI security analyst" inverted /></div>

      <div className="relative max-w-lg space-y-8">
        <div className="space-y-4">
          <h2 className="text-4xl font-semibold leading-[1.1] tracking-tight text-white xl:text-[2.75rem]">
            Alerts in.<br />
            <span className="bg-gradient-to-r from-sky-300 to-indigo-300 bg-clip-text text-transparent">Resolved cases out.</span>
          </h2>
          <p className="text-[15px] leading-relaxed text-white/65">
            Casewright triages every alert, builds the case around it, and turns confirmed attacks into new detection rules.
          </p>
        </div>

        <ol className="flex items-center gap-2" aria-label="How a case flows">
          {FLOW.map(({ Icon, label }, i) => (
            <li key={label} className="flex items-center gap-2">
              <span className="flex items-center gap-2 rounded-full border border-white/10 bg-white/5 py-1.5 pl-1.5 pr-3 text-xs font-medium text-white/85 backdrop-blur">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-white/10">
                  <Icon size={13} strokeWidth={2.25} aria-hidden="true" />
                </span>
                {label}
              </span>
              {i < FLOW.length - 1 && <ArrowRight size={14} className="text-white/30" aria-hidden="true" />}
            </li>
          ))}
        </ol>

        <ul className="space-y-5">
          {FEATURES.map(({ Icon, title, detail }) => (
            <li key={title} className="flex gap-4">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-white/10 bg-gradient-to-b from-white/10 to-white/[0.03] text-sky-300">
                <Icon size={17} strokeWidth={2} aria-hidden="true" />
              </span>
              <div>
                <p className="text-sm font-medium text-white">{title}</p>
                <p className="mt-0.5 text-[13px] leading-relaxed text-white/55">{detail}</p>
              </div>
            </li>
          ))}
        </ul>
      </div>

      <p className="relative text-xs text-white/45">Run it with Ollama and your logs never leave the host.</p>
    </aside>
  )
}

function LoginForm({ onSuccess }: { onSuccess: () => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [reveal, setReveal] = useState(false)
  const [capsLock, setCapsLock] = useState(false)
  const [busy, setBusy] = useState(false)
  const [failure, setFailure] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setFailure(null)
    try {
      await login(username, password)
      setPassword('')
      onSuccess()
    } catch (err) {
      setFailure(apiError(err))
    } finally {
      setBusy(false)
    }
  }

  const trackCaps = (e: KeyboardEvent<HTMLInputElement>) => setCapsLock(e.getModifierState?.('CapsLock') ?? false)

  return (
    <form onSubmit={submit} className="login-rise w-full max-w-[22rem] space-y-6" aria-labelledby="login-title" aria-busy={busy}>
      <div className="space-y-1.5">
        <h1 id="login-title" className="text-2xl font-semibold tracking-tight text-fg">Sign in to Casewright</h1>
        <p className="text-sm text-fg-muted">Welcome back. Pick up the cases where you left them.</p>
      </div>

      {failure && (
        <div role="alert" className="flex items-start gap-2.5 rounded-lg border border-danger/30 bg-danger/10 px-3 py-2.5 text-sm text-danger">
          <CircleAlert size={16} className="mt-0.5 shrink-0" aria-hidden="true" />
          <span>{failure}</span>
        </div>
      )}

      <div className="space-y-4">
        <Field id="login-username" label="Username" Icon={UserRound}>
          <input
            id="login-username"
            className="login-input"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            autoCapitalize="none"
            spellCheck={false}
            placeholder="analyst"
            autoFocus
            required
          />
        </Field>

        <Field
          id="login-password"
          label="Password"
          Icon={LockKeyhole}
          trailing={
            <button
              type="button"
              onClick={() => setReveal((r) => !r)}
              aria-label={reveal ? 'Hide password' : 'Show password'}
              aria-pressed={reveal}
              className="rounded-md p-1.5 text-fg-subtle transition-colors hover:bg-fg/6 hover:text-fg"
            >
              {reveal ? <EyeOff size={16} aria-hidden="true" /> : <Eye size={16} aria-hidden="true" />}
            </button>
          }
        >
          <input
            id="login-password"
            className="login-input pr-11"
            type={reveal ? 'text' : 'password'}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            onKeyDown={trackCaps}
            onKeyUp={trackCaps}
            onBlur={() => setCapsLock(false)}
            autoComplete="current-password"
            placeholder="••••••••••••"
            aria-describedby={capsLock ? 'login-caps' : undefined}
            required
          />
        </Field>
        {capsLock && (
          <p id="login-caps" className="-mt-2 text-xs text-warning">Caps Lock is on.</p>
        )}
      </div>

      <button
        type="submit"
        disabled={busy || !username || !password}
        className="group flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-accent text-sm font-medium text-on-accent shadow-card transition-all hover:bg-accent-strong hover:shadow-pop active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none"
      >
        {busy ? (
          <>
            <Spinner size={16} />
            Signing in…
          </>
        ) : (
          <>
            Sign in
            <ArrowRight size={16} className="transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
          </>
        )}
      </button>

      <p className="text-center text-xs text-fg-subtle">
        Forgot your password? An admin can reset it from <span className="font-medium text-fg-muted">Users</span>.
      </p>
    </form>
  )
}

function Field({ id, label, Icon, trailing, children }: { id: string; label: string; Icon: LucideIcon; trailing?: ReactNode; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="block text-xs font-medium text-fg-muted">{label}</label>
      <div className="relative">
        <Icon size={16} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-fg-subtle" aria-hidden="true" />
        {children}
        {trailing && <div className="absolute right-1.5 top-1/2 -translate-y-1/2">{trailing}</div>}
      </div>
    </div>
  )
}
