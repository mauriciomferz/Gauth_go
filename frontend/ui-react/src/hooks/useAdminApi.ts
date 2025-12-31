// React hooks for admin handler API calls
// Provides easy-to-use hooks with loading states and error handling

import { useState, useEffect, useCallback } from 'react';
import { apiFetch, apiPost, apiPut, apiDelete, ApiError, TenantError } from '../utils/api';
import type {
  PoAListResponse,
  CircuitBreakerListResponse,
  RateLimiterListResponse,
  RetryPolicyListResponse,
  EventListResponse,
  EventTypeListResponse,
  PolicyListResponse,
  VariableListResponse,
  FeatureFlagListResponse,
} from '../types/admin';

interface UseApiOptions {
  autoFetch?: boolean;
}

interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

// ============================================================================
// Generic API Hook
// ============================================================================

export function useApi<T>(
  url: string,
  options: UseApiOptions = { autoFetch: true }
) {
  const [state, setState] = useState<ApiState<T>>({
    data: null,
    loading: false,
    error: null,
  });

  const fetchData = useCallback(async () => {
    setState(prev => ({ ...prev, loading: true, error: null }));
    try {
      const response = await apiFetch(url);
      const data = await response.json();
      setState({ data, loading: false, error: null });
      return data;
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'An unexpected error occurred';
      setState({ data: null, loading: false, error: errorMessage });
      throw error;
    }
  }, [url]);

  useEffect(() => {
    if (options.autoFetch) {
      fetchData();
    }
  }, [options.autoFetch, fetchData]);

  return {
    ...state,
    refetch: fetchData,
  };
}

// ============================================================================
// Proof of Authorization Hooks
// ============================================================================

export function usePowerOfAttorneyList() {
  return useApi<PoAListResponse>('/api/admin/poa');
}

export function usePowerOfAttorney(id: string) {
  return useApi<any>(`/api/admin/poa/${id}`);
}

export function usePoAMutations() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createPoA = async (data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPost('/api/admin/poa', data);
      const result = await response.json();
      return result;
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to create PoA';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const updatePoA = async (id: string, data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPut(`/api/admin/poa/${id}`, data);
      const result = await response.json();
      return result;
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to update PoA';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const deletePoA = async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      await apiDelete(`/api/admin/poa/${id}`);
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to delete PoA';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  return {
    createPoA,
    updatePoA,
    deletePoA,
    loading,
    error,
  };
}

// ============================================================================
// Resilience Patterns Hooks
// ============================================================================

export function useCircuitBreakers() {
  return useApi<CircuitBreakerListResponse>('/api/admin/resilience/circuit-breakers');
}

export function useRateLimiters() {
  return useApi<RateLimiterListResponse>('/api/admin/resilience/rate-limiters');
}

export function useRetryPolicies() {
  return useApi<RetryPolicyListResponse>('/api/admin/resilience/retry-policies');
}

export function useResilienceMutations() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createCircuitBreaker = async (data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPost('/api/admin/resilience/circuit-breakers', data);
      return await response.json();
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to create circuit breaker';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const createRateLimiter = async (data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPost('/api/admin/resilience/rate-limiters', data);
      return await response.json();
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to create rate limiter';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const createRetryPolicy = async (data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPost('/api/admin/resilience/retry-policies', data);
      return await response.json();
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to create retry policy';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  return {
    createCircuitBreaker,
    createRateLimiter,
    createRetryPolicy,
    loading,
    error,
  };
}

// ============================================================================
// Event System Hooks
// ============================================================================

export function useEvents() {
  return useApi<EventListResponse>('/api/admin/events');
}

export function useEventTypes() {
  return useApi<EventTypeListResponse>('/api/admin/events/types');
}

// ============================================================================
// Authorization Engine Hooks
// ============================================================================

export function useAuthorizationPolicies() {
  return useApi<PolicyListResponse>('/api/admin/authz/policies');
}

export function useAuthzMutations() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createPolicy = async (data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPost('/api/admin/authz/policies', data);
      return await response.json();
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to create policy';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const updatePolicy = async (id: string, data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPut(`/api/admin/authz/policies/${id}`, data);
      return await response.json();
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to update policy';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const deletePolicy = async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      await apiDelete(`/api/admin/authz/policies/${id}`);
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to delete policy';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  return {
    createPolicy,
    updatePolicy,
    deletePolicy,
    loading,
    error,
  };
}

// ============================================================================
// Configuration Management Hooks
// ============================================================================

export function useConfigVariables() {
  return useApi<VariableListResponse>('/api/admin/config/variables');
}

export function useFeatureFlags() {
  return useApi<FeatureFlagListResponse>('/api/admin/config/feature-flags');
}

export function useConfigMutations() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createVariable = async (data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPost('/api/admin/config/variables', data);
      return await response.json();
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to create variable';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const createFeatureFlag = async (data: any) => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiPost('/api/admin/config/feature-flags', data);
      return await response.json();
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to create feature flag';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const deleteVariable = async (key: string) => {
    setLoading(true);
    setError(null);
    try {
      await apiDelete(`/api/admin/config/variables/${key}`);
    } catch (error) {
      const errorMessage = error instanceof ApiError || error instanceof TenantError
        ? error.message
        : 'Failed to delete variable';
      setError(errorMessage);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  return {
    createVariable,
    createFeatureFlag,
    deleteVariable,
    loading,
    error,
  };
}
