import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ApiError, refreshSession } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function OidcCallbackPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const setSession = useAuthStore((s) => s.setSession)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const token = searchParams.get('token')
    if (!token) {
      setError(t('login.oidcMissingToken'))
      return
    }

    let cancelled = false
    refreshSession(token)
      .then((data) => {
        if (cancelled) return
        setSession({
          token: data.token,
          userId: data.user_id,
          tenantId: data.tenant_id,
          username: data.username,
          displayName: data.display_name,
          permissions: data.permissions,
        })
        navigate('/dashboard', { replace: true })
      })
      .catch((err) => {
        if (cancelled) return
        if (err instanceof ApiError && err.status === 401) {
          setError(t('login.oidcUnauthorized'))
        } else {
          setError(t('login.oidcFailed'))
        }
      })

    return () => {
      cancelled = true
    }
  }, [navigate, searchParams, setSession, t])

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t('login.oidcSigningIn')}</CardTitle>
        </CardHeader>
        <CardContent>
          {error ? (
            <p className="text-sm text-destructive">{error}</p>
          ) : (
            <p className="text-sm text-muted-foreground">{t('login.oidcPleaseWait')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
