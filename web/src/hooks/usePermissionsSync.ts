import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { ApiError, fetchPermissions, refreshSession } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'

function samePermissions(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false
  const left = [...a].sort()
  const right = [...b].sort()
  return left.every((code, index) => code === right[index])
}

export function usePermissionsSync() {
  const token = useAuthStore((s) => s.token)
  const storedPermissions = useAuthStore((s) => s.permissions)
  const setPermissions = useAuthStore((s) => s.setPermissions)
  const updateSessionToken = useAuthStore((s) => s.updateSessionToken)
  const queryClient = useQueryClient()
  const refreshingRef = useRef(false)

  const query = useQuery({
    queryKey: ['permissions', token],
    queryFn: () => fetchPermissions(token!),
    enabled: Boolean(token),
    staleTime: 60_000,
    retry: (failureCount, error) => {
      if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
        return false
      }
      return failureCount < 2
    },
  })

  useEffect(() => {
    if (!token || !query.isError) return
    const err = query.error
    if (!(err instanceof ApiError)) return
    if (refreshingRef.current) return

    refreshingRef.current = true
    refreshSession(token)
      .then((session) => {
        updateSessionToken(session.token, session.permissions)
        queryClient.invalidateQueries({ queryKey: ['permissions'] })
      })
      .catch(() => {
        useAuthStore.getState().clearSession()
      })
      .finally(() => {
        refreshingRef.current = false
      })
  }, [query.isError, query.error, token, updateSessionToken, queryClient])

  useEffect(() => {
    const latest = query.data?.permissions
    if (!latest || !token) return

    setPermissions(latest)

    if (samePermissions(latest, storedPermissions) || refreshingRef.current) {
      return
    }

    refreshingRef.current = true
    refreshSession(token)
      .then((session) => {
        updateSessionToken(session.token, session.permissions)
        queryClient.invalidateQueries({ queryKey: ['workflows'] })
        queryClient.invalidateQueries({ queryKey: ['search'] })
        queryClient.invalidateQueries({ queryKey: ['notifications'] })
      })
      .catch(() => {
        // Keep UI permissions in sync; user can sign out and back in if refresh fails.
      })
      .finally(() => {
        refreshingRef.current = false
      })
  }, [
    query.data?.permissions,
    token,
    storedPermissions,
    setPermissions,
    updateSessionToken,
    queryClient,
  ])

  return query
}
