import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import toast from 'react-hot-toast';

class ApiService {
  private api: AxiosInstance;
  private baseURL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

  constructor() {
    this.api = axios.create({
      baseURL: `${this.baseURL}/api/v1`,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    // Request interceptor - add auth token
    this.api.interceptors.request.use(
      (config) => {
        const token = this.getStoredToken();
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor - handle errors and token refresh
    this.api.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;

        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;

          try {
            await this.refreshToken();
            const newToken = this.getStoredToken();
            originalRequest.headers.Authorization = `Bearer ${newToken}`;
            return this.api(originalRequest);
          } catch (refreshError) {
            this.handleAuthError();
            return Promise.reject(refreshError);
          }
        }

        this.handleError(error);
        return Promise.reject(error);
      }
    );
  }

  private getStoredToken(): string | null {
    const authData = localStorage.getItem('gauth-auth-store');
    if (authData) {
      const parsed = JSON.parse(authData);
      return parsed?.state?.token?.accessToken || null;
    }
    return null;
  }

  private async refreshToken(): Promise<void> {
    const authData = localStorage.getItem('gauth-auth-store');
    if (!authData) throw new Error('No refresh token available');

    const parsed = JSON.parse(authData);
    const refreshToken = parsed?.state?.token?.refreshToken;
    
    if (!refreshToken) throw new Error('No refresh token available');

    const response = await axios.post(`${this.baseURL}/api/v1/auth/refresh`, {}, {
      headers: { Authorization: `Bearer ${refreshToken}` }
    });

    // Update stored token
    const newTokenData = {
      ...parsed.state.token,
      accessToken: response.data.access_token,
      refreshToken: response.data.refresh_token,
      expiresAt: response.data.expires_at,
    };

    localStorage.setItem('gauth-auth-store', JSON.stringify({
      ...parsed,
      state: { ...parsed.state, token: newTokenData }
    }));
  }

  private handleAuthError() {
    localStorage.removeItem('gauth-auth-store');
    toast.error('Session expired. Please login again.');
    window.location.href = '/login';
  }

  private handleError(error: any) {
    if (error.response?.status >= 500) {
      toast.error('Server error. Please try again later.');
    } else if (error.response?.status === 403) {
      toast.error('Access denied. Insufficient permissions.');
    } else if (error.code === 'NETWORK_ERROR') {
      toast.error('Network error. Please check your connection.');
    }
  }

  // Authentication endpoints
  async login(credentials: { email: string; password: string }) {
    const response = await this.api.post('/auth/login', credentials);
    return response.data;
  }

  async logout() {
    try {
      await this.api.post('/auth/logout');
    } catch (error) {
      // Continue with logout even if request fails
    }
  }

  async refreshUserToken() {
    const response = await this.api.post('/auth/refresh');
    return response.data;
  }

  // Token management endpoints
  async getTokens(params?: { page?: number; limit?: number; status?: string }) {
    const response = await this.api.get('/tokens', { params });
    return response.data;
  }

  async createToken(data: { 
    claims: Record<string, any>; 
    duration: string; 
    scope?: string[] 
  }) {
    // Transform the data to match backend expected format
    const backendData = {
      type: "JWT", // Default token type
      subject: data.claims.sub || data.claims.client_id || "anonymous",
      scopes: data.scope || [],
      claims: data.claims,
      expires_in: this.parseDurationToSeconds(data.duration),
    };
    
    const response = await this.api.post('/tokens', backendData);
    return response.data;
  }

  private parseDurationToSeconds(duration: string): number {
    // Convert duration string like "1h", "30m", "24h" to seconds
    const units: Record<string, number> = {
      's': 1,
      'm': 60,
      'h': 3600,
      'd': 86400
    };
    
    const match = duration.match(/^(\d+)([smhd])$/);
    if (!match) {
      return 3600; // Default 1 hour in seconds
    }
    
    const [, value, unit] = match;
    return parseInt(value) * (units[unit] || 3600);
  }

  async revokeToken(tokenId: string) {
    const response = await this.api.delete(`/tokens/${tokenId}`);
    return response.data;
  }

  async validateToken(token: string) {
    const response = await this.api.post('/tokens/validate', { token });
    return response.data;
  }

  // Compliance endpoints
  async getComplianceStatus() {
    const response = await this.api.get('/compliance/status');
    return response.data;
  }

  async getAuditLogs(params?: { 
    startDate?: string; 
    endDate?: string; 
    action?: string;
    page?: number;
    limit?: number;
  }) {
    const response = await this.api.get('/audit/logs', { params });
    return response.data;
  }

  // Power of Attorney endpoints
  async createPowerOfAttorney(data: {
    grantor: string;
    grantee: string;
    powers: string[];
    restrictions?: string[];
    expiresAt?: string;
  }) {
    const response = await this.api.post('/power-of-attorney', data);
    return response.data;
  }

  async getPowerOfAttorneys(userId?: string) {
    const response = await this.api.get('/power-of-attorney', {
      params: userId ? { user_id: userId } : {}
    });
    return response.data;
  }

  async revokePowerOfAttorney(id: string) {
    const response = await this.api.delete(`/power-of-attorney/${id}`);
    return response.data;
  }

  // RFC implementations
  async getRFC111Status() {
    const response = await this.api.get('/rfc111/status');
    return response.data;
  }

  async getRFC115Status() {
    const response = await this.api.get('/rfc115/status');
    return response.data;
  }

  // Metrics endpoints
  async getSystemMetrics() {
    const response = await this.api.get('/metrics/system');
    return response.data;
  }

  async getTokenMetrics() {
    const response = await this.api.get('/metrics/tokens');
    return response.data;
  }

  // Generic request method
  async request<T>(config: AxiosRequestConfig): Promise<T> {
    const response = await this.api.request<T>(config);
    return response.data;
  }
}

export default new ApiService();
