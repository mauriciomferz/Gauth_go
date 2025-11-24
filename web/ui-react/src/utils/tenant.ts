// Tenant management utility for admin handlers
// Handles tenant ID storage and retrieval for multi-tenant API calls

/**
 * Get the current tenant ID from localStorage
 * Returns default tenant if not set
 */
export const getTenantId = (): string => {
  // Try to get from localStorage first
  const storedTenant = localStorage.getItem('tenant_id');
  if (storedTenant) return storedTenant;
  
  // Try to get from environment variable
  const envTenant = import.meta.env.VITE_DEFAULT_TENANT_ID;
  if (envTenant) {
    localStorage.setItem('tenant_id', envTenant);
    return envTenant;
  }
  
  // Default tenant for development
  const defaultTenant = 'test-tenant-1';
  localStorage.setItem('tenant_id', defaultTenant);
  return defaultTenant;
};

/**
 * Set the tenant ID in localStorage
 */
export const setTenantId = (tenantId: string): void => {
  if (!tenantId || tenantId.trim() === '') {
    throw new Error('Tenant ID cannot be empty');
  }
  localStorage.setItem('tenant_id', tenantId);
};

/**
 * Clear the stored tenant ID
 */
export const clearTenantId = (): void => {
  localStorage.removeItem('tenant_id');
};

/**
 * Add tenant_id parameter to a URL
 * Handles URLs with or without existing query parameters
 */
export const addTenantParam = (url: string): string => {
  const tenantId = getTenantId();
  const separator = url.includes('?') ? '&' : '?';
  return `${url}${separator}tenant_id=${encodeURIComponent(tenantId)}`;
};

/**
 * Check if a tenant ID is currently set
 */
export const hasTenantId = (): boolean => {
  return localStorage.getItem('tenant_id') !== null;
};

/**
 * Get all available tenants (for tenant switcher UI)
 * This would typically come from an API call
 */
export const getAvailableTenants = (): string[] => {
  const storedTenants = localStorage.getItem('available_tenants');
  if (storedTenants) {
    try {
      return JSON.parse(storedTenants);
    } catch {
      return ['test-tenant-1'];
    }
  }
  return ['test-tenant-1'];
};

/**
 * Set available tenants in localStorage
 */
export const setAvailableTenants = (tenants: string[]): void => {
  localStorage.setItem('available_tenants', JSON.stringify(tenants));
};
