// WebSocket Manager for Real-time Updates
// Handles connections to backend event streams

import { useEffect, useState } from 'react'

interface EventMessage {
  type: string
  data: any
  timestamp: string
}

interface MetricsUpdate {
  systemMetrics?: any
  componentHealth?: any
  performanceHistory?: any
  tokenViolations?: any
  semanticCounters?: any
}

type EventCallback = (message: EventMessage) => void

class WebSocketManager {
  private eventSource: EventSource | null = null
  private listeners: Map<string, Set<EventCallback>> = new Map()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000

  connect(url: string = '/api/v1/events/stream') {
    if (this.eventSource) {
      this.disconnect()
    }

    try {
      this.eventSource = new EventSource(url)

      this.eventSource.onopen = () => {
        console.log('[WebSocket] Connected to event stream')
        this.reconnectAttempts = 0
      }

      this.eventSource.onmessage = (event) => {
        try {
          const message: EventMessage = JSON.parse(event.data)
          this.notifyListeners(message.type, message)
          this.notifyListeners('*', message) // Wildcard listeners
        } catch (error) {
          console.error('[WebSocket] Failed to parse message:', error)
        }
      }

      this.eventSource.onerror = (error) => {
        console.error('[WebSocket] Connection error:', error)
        this.handleReconnect(url)
      }
    } catch (error) {
      console.error('[WebSocket] Failed to connect:', error)
      this.handleReconnect(url)
    }
  }

  disconnect() {
    if (this.eventSource) {
      this.eventSource.close()
      this.eventSource = null
      console.log('[WebSocket] Disconnected')
    }
  }

  private handleReconnect(url: string) {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
      console.log(`[WebSocket] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`)
      
      setTimeout(() => {
        this.connect(url)
      }, delay)
    } else {
      console.error('[WebSocket] Max reconnection attempts reached')
    }
  }

  on(eventType: string, callback: EventCallback) {
    if (!this.listeners.has(eventType)) {
      this.listeners.set(eventType, new Set())
    }
    this.listeners.get(eventType)!.add(callback)
  }

  off(eventType: string, callback: EventCallback) {
    const listeners = this.listeners.get(eventType)
    if (listeners) {
      listeners.delete(callback)
      if (listeners.size === 0) {
        this.listeners.delete(eventType)
      }
    }
  }

  private notifyListeners(eventType: string, message: EventMessage) {
    const listeners = this.listeners.get(eventType)
    if (listeners) {
      listeners.forEach(callback => {
        try {
          callback(message)
        } catch (error) {
          console.error(`[WebSocket] Listener error for ${eventType}:`, error)
        }
      })
    }
  }

  isConnected(): boolean {
    return this.eventSource !== null && this.eventSource.readyState === EventSource.OPEN
  }

  getConnectionState(): 'connecting' | 'open' | 'closed' {
    if (!this.eventSource) return 'closed'
    
    switch (this.eventSource.readyState) {
      case EventSource.CONNECTING:
        return 'connecting'
      case EventSource.OPEN:
        return 'open'
      case EventSource.CLOSED:
        return 'closed'
      default:
        return 'closed'
    }
  }
}

// Singleton instance
export const wsManager = new WebSocketManager()

// React hook for WebSocket events
export function useWebSocketEvent(eventType: string, callback: EventCallback) {
  const [isConnected, setIsConnected] = useState(wsManager.isConnected())

  useEffect(() => {
    // Connect if not already connected
    if (!wsManager.isConnected()) {
      wsManager.connect()
    }

    // Register callback
    wsManager.on(eventType, callback)

    // Update connection status
    const checkConnection = () => {
      setIsConnected(wsManager.isConnected())
    }
    const interval = setInterval(checkConnection, 1000)

    // Cleanup
    return () => {
      wsManager.off(eventType, callback)
      clearInterval(interval)
    }
  }, [eventType, callback])

  return { isConnected }
}

// React hook for metrics updates
export function useMetricsStream(callback: (metrics: MetricsUpdate) => void) {
  return useWebSocketEvent('metrics', (message) => {
    if (message.data) {
      callback(message.data as MetricsUpdate)
    }
  })
}

// Export types
export type { EventMessage, EventCallback, MetricsUpdate }
