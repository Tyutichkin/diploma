import { createContext, useContext } from 'react';
import { AuthSession } from '../types/auth';

export interface AuthContextValue {
  session: AuthSession | null;
  setSession: (session: AuthSession | null) => void;
  logout: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth() {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error('useAuth must be used inside AuthContext provider');
  }

  return context;
}
