// API utility for making tenant-aware API calls to admin handlers
// Automatically adds tenant_id parameter and authorization headers

import { getTenantId } from './tenant';

export class TenantError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'TenantError';
  }
}

export class ApiError extends Error {
  status: number;
  data?: any;

  constructor(message: string, status: number, data?: any) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

interface FetchOptions extends RequestInit {
  skipTenant?: boolean;
  skipAuth?: boolean;
}

/**
 * Enhanced fetch wrapper that:
 * - Automatically adds tenant_id parameter
 * - Adds authorization header if token exists
 * - Handles common errors
 * - Provides better error messages
 */
export const apiFetch = async (
  url: string,
  options: FetchOptions = {}
): Promise<Response> => {
  const { skipTenant, skipAuth, ...fetchOptions } = options;

  // Add tenant_id parameter unless explicitly skipped
  let finalUrl = url;
  if (!skipTenant) {
    const tenantId = getTenantId();
    const separator = url.includes('?') ? '&' : '?';
    finalUrl = `${url}${separator}tenant_id=${encodeURIComponent(tenantId)}`;
  }

  // Add authorization header if token exists and not skipped
  const token = !skipAuth ? localStorage.getItem('admin_token') : null;
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` }),
    ...fetchOptions.headers,
  };

  try {
    const response = await fetch(finalUrl, {
      ...fetchOptions,
      headers,
    });

    // Handle tenant-specific errors
    if (response.status === 403) {
      const error = await response.json().catch(() => ({}));
      throw new TenantError(error.message || 'Access denied for this tenant');
    }

    if (response.status === 400) {
      const error = await response.json().catch(() => ({}));
      if (error.message?.includes('tenant')) {
        throw new TenantError(error.message);
      }
      throw new ApiError(error.message || 'Bad request', 400, error);
    }

    if (response.status === 401) {
      // Token expired or invalid
      localStorage.removeItem('admin_token');
      throw new ApiError('Authentication required', 401);
    }

    if (response.status === 404) {
      const error = await response.json().catch(() => ({}));
      throw new ApiError(error.message || 'Resource not found', 404, error);
    }

    if (response.status === 500) {
      const error = await response.json().catch(() => ({}));
      throw new ApiError(error.message || 'Internal server error', 500, error);
    }

    return response;
  } catch (error) {
    // Re-throw our custom errors
    if (error instanceof TenantError || error instanceof ApiError) {
      throw error;
    }

    // Handle network errors
    if (error instanceof TypeError && error.message === 'Failed to fetch') {
      throw new ApiError('Network error: Cannot connect to server', 0);
    }

    // Unknown error
    throw new ApiError('An unexpected error occurred', 0, error);
  }
};

/**
 * Convenience method for GET requests
 */
export const apiGet = async (url: string, options?: FetchOptions): Promise<Response> => {
  return apiFetch(url, { ...options, method: 'GET' });
};

/**
 * Convenience method for POST requests
 */
export const apiPost = async (
  url: string,
  data: any,
  options?: FetchOptions
): Promise<Response> => {
  return apiFetch(url, {
    ...options,
    method: 'POST',
    body: JSON.stringify(data),
  });
};

/**
 * Convenience method for PUT requests
 */
export const apiPut = async (
  url: string,
  data: any,
  options?: FetchOptions
): Promise<Response> => {
  return apiFetch(url, {
    ...options,
    method: 'PUT',
    body: JSON.stringify(data),
  });
};

/**
 * Convenience method for DELETE requests
 */
export const apiDelete = async (url: string, options?: FetchOptions): Promise<Response> => {
  return apiFetch(url, { ...options, method: 'DELETE' });
};

/**
 * Parse JSON response safely
 */
export const parseResponse = async <T>(response: Response): Promise<T> => {
  try {
    return await response.json();
  } catch (_error) {
    throw new ApiError('Failed to parse response', response.status);
  }
};

/**
 * Combined fetch and parse
 */
export const fetchJson = async <T>(url: string, options?: FetchOptions): Promise<T> => {
  const response = await apiFetch(url, options);
  return parseResponse<T>(response);
};
