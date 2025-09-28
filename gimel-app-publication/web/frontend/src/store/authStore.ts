import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

export interface User {
  id: string;
  email: string;
  name: string;
  roles: string[];
  permissions: string[];
}

export interface TokenData {
  accessToken: string;
  refreshToken: string;
  expiresAt: Date;
  tokenType: string;
}

interface AuthState {
  user: User | null;
  token: TokenData | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  
  // Actions
  setUser: (user: User) => void;
  setToken: (token: TokenData) => void;
  login: (credentials: { email: string; password: string }) => Promise<void>;
  logout: () => void;
  refreshToken: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthState>()(
  devtools(
    persist(
      (set, get) => ({
        user: null,
        token: null,
        isAuthenticated: false,
        isLoading: false,
        error: null,

        setUser: (user) => set({ user, isAuthenticated: !!user }),
        
        setToken: (token) => set({ token }),

        login: async (credentials) => {
          set({ isLoading: true, error: null });
          try {
            const response = await fetch('/api/v1/auth/login', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify(credentials),
            });

            if (!response.ok) {
              throw new Error('Login failed');
            }

            const data = await response.json();
            const user: User = data.user;
            const token: TokenData = {
              accessToken: data.access_token,
              refreshToken: data.refresh_token,
              expiresAt: new Date(data.expires_at),
              tokenType: data.token_type || 'Bearer',
            };

            set({ 
              user, 
              token, 
              isAuthenticated: true, 
              isLoading: false 
            });
          } catch (error) {
            set({ 
              error: error instanceof Error ? error.message : 'Login failed',
              isLoading: false 
            });
          }
        },

        logout: () => {
          set({ 
            user: null, 
            token: null, 
            isAuthenticated: false,
            error: null 
          });
        },

        refreshToken: async () => {
          const { token } = get();
          if (!token?.refreshToken) return;

          try {
            const response = await fetch('/api/v1/auth/refresh', {
              method: 'POST',
              headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token.refreshToken}`,
              },
            });

            if (!response.ok) {
              throw new Error('Token refresh failed');
            }

            const data = await response.json();
            const newToken: TokenData = {
              accessToken: data.access_token,
              refreshToken: data.refresh_token,
              expiresAt: new Date(data.expires_at),
              tokenType: data.token_type || 'Bearer',
            };

            set({ token: newToken });
          } catch (error) {
            // If refresh fails, logout user
            get().logout();
          }
        },

        clearError: () => set({ error: null }),
      }),
      {
        name: 'gauth-auth-store',
        partialize: (state) => ({ 
          user: state.user, 
          token: state.token,
          isAuthenticated: state.isAuthenticated,
        }),
      }
    )
  )
);
