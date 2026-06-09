import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { ApiError, fetchOidcConfig, login, startOidcLogin } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { LanguageSwitcher } from '@/components/layout/LanguageSwitcher'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const setSession = useAuthStore((s) => s.setSession)
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const sessionExpired =
    (location.state as { reason?: string } | null)?.reason === 'session_expired'
  const [error, setError] = useState<string | null>(
    sessionExpired ? t('login.sessionExpired') : null,
  )
  const [loading, setLoading] = useState(false)
  const [oidcEnabled, setOidcEnabled] = useState(false)

  useEffect(() => {
    fetchOidcConfig()
      .then((cfg) => setOidcEnabled(cfg.enabled))
      .catch(() => setOidcEnabled(false))
  }, [])

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const data = await login({ username, password })
      setSession({
        token: data.token,
        userId: data.user_id,
        tenantId: data.tenant_id,
        username: data.username,
        displayName: data.display_name,
        permissions: data.permissions,
      })
      navigate('/dashboard', { replace: true })
    } catch (err) {
      if (err instanceof ApiError && err.status === 429) {
        setError(t('login.tooManyRequests'))
      } else if (err instanceof ApiError && err.status === 401) {
        setError(t('login.invalidCredentials'))
      } else {
        setError(t('login.gatewayUnavailable'))
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      <div className="mb-4">
        <LanguageSwitcher />
      </div>
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t('login.title')}</CardTitle>
          <CardDescription>{t('login.subtitle')}</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-2">
              <Label htmlFor="username">{t('login.username')}</Label>
              <Input
                id="username"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t('login.password')}</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            <Button className="w-full" type="submit" disabled={loading}>
              {loading ? t('login.submitting') : t('login.submit')}
            </Button>
            {oidcEnabled ? (
              <Button
                className="w-full"
                type="button"
                variant="outline"
                disabled={loading}
                onClick={() => startOidcLogin()}
              >
                {t('login.sso')}
              </Button>
            ) : null}
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
