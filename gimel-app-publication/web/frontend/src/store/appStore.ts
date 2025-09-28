import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

export interface AppNotification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message: string;
  timestamp: Date;
  read: boolean;
}

export interface AppMetrics {
  activeUsers: number;
  totalTransactions: number;
  successRate: number;
  averageResponseTime: number;
  lastUpdated: Date;
}

interface AppState {
  // UI State
  sidebarOpen: boolean;
  currentPage: string;
  theme: 'light' | 'dark';
  
  // Notifications
  notifications: AppNotification[];
  unreadCount: number;
  
  // Real-time metrics
  metrics: AppMetrics | null;
  isConnected: boolean;
  
  // Loading states
  loadingStates: Record<string, boolean>;
  
  // Actions
  toggleSidebar: () => void;
  setCurrentPage: (page: string) => void;
  toggleTheme: () => void;
  
  // Notifications
  addNotification: (notification: Omit<AppNotification, 'id' | 'timestamp' | 'read'>) => void;
  markNotificationRead: (id: string) => void;
  clearNotifications: () => void;
  
  // Metrics
  updateMetrics: (metrics: AppMetrics) => void;
  setConnectionStatus: (connected: boolean) => void;
  
  // Loading
  setLoading: (key: string, loading: boolean) => void;
  isLoading: (key: string) => boolean;
}

export const useAppStore = create<AppState>()(
  devtools(
    (set, get) => ({
      // UI State
      sidebarOpen: true,
      currentPage: 'dashboard',
      theme: 'light',
      
      // Notifications
      notifications: [],
      unreadCount: 0,
      
      // Metrics
      metrics: null,
      isConnected: false,
      
      // Loading states
      loadingStates: {},

      // UI Actions
      toggleSidebar: () => set((state) => ({ 
        sidebarOpen: !state.sidebarOpen 
      })),

      setCurrentPage: (page) => set({ currentPage: page }),

      toggleTheme: () => set((state) => ({ 
        theme: state.theme === 'light' ? 'dark' : 'light' 
      })),

      // Notification Actions
      addNotification: (notificationData) => {
        const notification: AppNotification = {
          ...notificationData,
          id: Math.random().toString(36).substr(2, 9),
          timestamp: new Date(),
          read: false,
        };

        set((state) => ({
          notifications: [notification, ...state.notifications].slice(0, 50), // Keep max 50 notifications
          unreadCount: state.unreadCount + 1,
        }));
      },

      markNotificationRead: (id) => set((state) => ({
        notifications: state.notifications.map(n => 
          n.id === id ? { ...n, read: true } : n
        ),
        unreadCount: Math.max(0, state.unreadCount - 1),
      })),

      clearNotifications: () => set({
        notifications: [],
        unreadCount: 0,
      }),

      // Metrics Actions
      updateMetrics: (metrics) => set({ 
        metrics: { ...metrics, lastUpdated: new Date() }
      }),

      setConnectionStatus: (connected) => set({ isConnected: connected }),

      // Loading Actions
      setLoading: (key, loading) => set((state) => ({
        loadingStates: { ...state.loadingStates, [key]: loading }
      })),

      isLoading: (key) => get().loadingStates[key] || false,
    })
  )
);
