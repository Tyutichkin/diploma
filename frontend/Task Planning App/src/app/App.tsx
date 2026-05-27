import { useCallback, useMemo, useState } from 'react';
import { RouterProvider } from 'react-router';
import { AuthPage } from './components/AuthPage';
import { Toaster } from './components/ui/sonner';
import { createRouter } from './routes.tsx';
import { AuthContext } from './context/auth-context';
import { logoutRequest, readStoredSession, writeStoredSession } from './api/client';
import { AuthSession } from './types/auth';

export default function App() {
  const [session, setSessionState] = useState<AuthSession | null>(() => readStoredSession());

  const setSession = useCallback((nextSession: AuthSession | null) => {
    setSessionState(nextSession);
    writeStoredSession(nextSession);
  }, []);

  const handleLogout = useCallback(async () => {
    if (session?.refreshToken) {
      try {
        await logoutRequest(session.refreshToken);
      } catch {
        // Локальную сессию всё равно чистим, даже если бэкенд не ответил
      }
    }

    setSession(null);
  }, [session, setSession]);

  const router = useMemo(() => createRouter(), []);
  const authContextValue = useMemo(
    () => ({
      session,
      setSession,
      logout: handleLogout,
    }),
    [handleLogout, session, setSession],
  );

  return (
    <AuthContext.Provider value={authContextValue}>
      {!session ? <AuthPage /> : <RouterProvider router={router} />}
      <Toaster position="top-right" expand visibleToasts={5} />
    </AuthContext.Provider>
  );
}
