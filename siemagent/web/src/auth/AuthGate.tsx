import { useEffect, useMemo } from 'react'
import type { ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import axios from 'axios'
import { RotateCw, ServerCrash } from 'lucide-react'
import { SESSION_EXPIRED_EVENT, getMe, logout } from '../lib/api'
import { BrandMark } from '../components/BrandMark'
import { BootSplash, Spinner } from '../components/Loading'
import { LoginPage } from './LoginPage'
import { AuthContext } from './auth'
import type { AuthContextValue, Permission } from './auth'

// AuthGate shows the login form until the server recognises the caller,
// then provides who they are and what they may do to the app.
export function AuthGate({ children }: { children: ReactNode }) {
  const qc = useQueryClient()
  const me = useQuery({
    queryKey: ['me'],
    queryFn: getMe,
    retry: false,
    staleTime: 60_000,
    // While the backend is down (still compiling, crashed, restarting), keep
    // checking so the app recovers by itself once it answers.
    refetchInterval: (q) => (q.state.error && !isUnauthorized(q.state.error) ? 3000 : false),
  })

  useEffect(() => {
    const onExpired = () => { void qc.invalidateQueries({ queryKey: ['me'] }) }
    window.addEventListener(SESSION_EXPIRED_EVENT, onExpired)
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, onExpired)
  }, [qc])

  const value = useMemo<AuthContextValue | null>(() => {
    if (!me.data) return null
    return {
      me: me.data,
      can: (p: Permission) => me.data.permissions[p],
      signOut: () => {
        // Reset drops every cached response (nothing from the old session
        // stays on screen) and refetches "me", which now shows the login.
        void logout().finally(() => { void qc.resetQueries() })
      },
    }
  }, [me.data, qc])

  if (me.isLoading) {
    return <BootSplash />
  }
  if (me.error && !isUnauthorized(me.error)) {
    return <ServerUnavailable error={me.error} onRetry={() => void me.refetch()} retrying={me.isFetching} />
  }
  // A 401 always means "log in", even though the previous user's data is
  // still cached from before the session expired.
  if (isUnauthorized(me.error) || !value) {
    return <LoginPage onSuccess={() => qc.invalidateQueries({ queryKey: ['me'] })} />
  }
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

function isUnauthorized(err: unknown): boolean {
  return axios.isAxiosError(err) && err.response?.status === 401
}

// describe explains why the backend didn't answer, in terms of what to check.
function describe(err: unknown): { title: string; detail: string } {
  const status = axios.isAxiosError(err) ? err.response?.status : undefined
  if (status === 404) {
    return {
      title: 'The backend is running an older version',
      detail: 'It does not know /api/auth/me. Stop the backend and start it again from the latest code (make dev).',
    }
  }
  if (status && status < 500) {
    return { title: `The backend answered ${status}`, detail: 'Check the backend terminal for the error.' }
  }
  return {
    title: 'Cannot reach the Casewright backend',
    detail:
      'The dashboard is up but the Go server on port 8080 is not answering. It may still be starting, or it stopped with an error: check the terminal running make dev (look for a line starting with "error:").',
  }
}

function ServerUnavailable({ error, onRetry, retrying }: { error: unknown; onRetry: () => void; retrying: boolean }) {
  const { title, detail } = describe(error)
  return (
    <main className="flex min-h-screen items-center justify-center bg-canvas p-4">
      <section role="alert" aria-labelledby="down-title" className="login-rise w-full max-w-md overflow-hidden rounded-2xl border border-line bg-surface shadow-pop">
        <div className="flex items-center gap-3 border-b border-line bg-surface-2 px-6 py-4">
          <BrandMark size={28} />
          <p className="text-sm font-semibold tracking-tight text-fg">Casewright</p>
        </div>
        <div className="space-y-4 p-6">
          <div className="flex items-start gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-warning/10 text-warning">
              <ServerCrash size={20} aria-hidden="true" />
            </span>
            <div className="space-y-1">
              <h1 id="down-title" className="text-base font-semibold text-fg">{title}</h1>
              <p className="text-sm leading-relaxed text-fg-muted">{detail}</p>
            </div>
          </div>
          <div className="flex items-center justify-between gap-3 rounded-lg bg-fg/4 px-3 py-2.5">
            <p className="flex items-center gap-2 text-xs text-fg-subtle">
              <span className="relative flex h-2 w-2" aria-hidden="true">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-warning opacity-60" />
                <span className="relative inline-flex h-2 w-2 rounded-full bg-warning" />
              </span>
              Retrying automatically every few seconds.
            </p>
            <button
              onClick={onRetry}
              disabled={retrying}
              className="flex shrink-0 items-center gap-1.5 rounded-lg border border-line-strong bg-surface px-3 py-1.5 text-sm text-fg transition-colors hover:bg-fg/4 disabled:opacity-50"
            >
              {retrying ? <Spinner size={13} /> : <RotateCw size={13} aria-hidden="true" />}
              {retrying ? 'Checking…' : 'Retry now'}
            </button>
          </div>
        </div>
      </section>
    </main>
  )
}
