import { useAppStore } from '../store/appStore';

export interface WebSocketMessage {
  type: 'token_created' | 'token_revoked' | 'token_expired' | 'audit_event' | 'metrics_update' | 'compliance_alert';
  data: any;
  timestamp: string;
}

class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 1000;
  private url = (window as any).REACT_APP_WS_URL || 'ws://localhost:8080/ws';
  private isIntentionallyClosed = false;

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.isIntentionallyClosed = false;
    
    try {
      this.ws = new WebSocket(this.url);
      this.setupEventHandlers();
    } catch (error) {
      console.error('WebSocket connection failed:', error);
      this.scheduleReconnect();
    }
  }

  private setupEventHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = (event) => {
      console.log('WebSocket connected');
      useAppStore.getState().setConnectionStatus(true);
      this.reconnectAttempts = 0;
      
      // Send authentication token
      const token = this.getAuthToken();
      if (token) {
        this.send({ type: 'auth', token });
      }
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    this.ws.onclose = (event) => {
      console.log('WebSocket disconnected:', event.code, event.reason);
      useAppStore.getState().setConnectionStatus(false);
      
      if (!this.isIntentionallyClosed && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      useAppStore.getState().setConnectionStatus(false);
    };
  }

  private handleMessage(message: WebSocketMessage): void {
    const appStore = useAppStore.getState();
    
    switch (message.type) {
      case 'token_created':
        appStore.addNotification({
          type: 'success',
          title: 'Token Created',
          message: `New token created for ${message.data.owner_id}`,
        });
        break;

      case 'token_revoked':
        appStore.addNotification({
          type: 'warning',
          title: 'Token Revoked',
          message: `Token for ${message.data.owner_id} has been revoked`,
        });
        break;

      case 'token_expired':
        appStore.addNotification({
          type: 'info',
          title: 'Token Expired',
          message: `Token for ${message.data.owner_id} has expired`,
        });
        break;

      case 'audit_event':
        appStore.addNotification({
          type: 'info',
          title: 'Audit Event',
          message: `${message.data.action} performed by ${message.data.user_id}`,
        });
        break;

      case 'metrics_update':
        appStore.updateMetrics(message.data);
        break;

      case 'compliance_alert':
        appStore.addNotification({
          type: 'error',
          title: 'Compliance Alert',
          message: message.data.message,
        });
        break;

      default:
        console.log('Unknown message type:', message.type);
    }
  }

  private scheduleReconnect(): void {
    if (this.isIntentionallyClosed) return;

    const delay = Math.min(
      this.reconnectInterval * Math.pow(2, this.reconnectAttempts),
      30000
    );

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts + 1})`);
    
    setTimeout(() => {
      this.reconnectAttempts++;
      this.connect();
    }, delay);
  }

  private getAuthToken(): string | null {
    const authData = localStorage.getItem('gauth-auth-store');
    if (authData) {
      const parsed = JSON.parse(authData);
      return parsed?.state?.token?.accessToken || null;
    }
    return null;
  }

  send(data: any): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    } else {
      console.warn('WebSocket is not connected');
    }
  }

  subscribe(eventType: string, callback: (data: any) => void): () => void {
    // This is a simplified subscription system
    // In a real implementation, you might want a more sophisticated event system
    const handleMessage = (event: MessageEvent) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        if (message.type === eventType) {
          callback(message.data);
        }
      } catch (error) {
        console.error('Failed to parse message in subscription:', error);
      }
    };

    this.ws?.addEventListener('message', handleMessage);

    return () => {
      this.ws?.removeEventListener('message', handleMessage);
    };
  }

  disconnect(): void {
    this.isIntentionallyClosed = true;
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    useAppStore.getState().setConnectionStatus(false);
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

export default new WebSocketService();