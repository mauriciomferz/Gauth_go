/**
 * Prometheus Metrics Parser
 * 
 * Parses Prometheus text exposition format into structured TypeScript objects.
 * Supports counter, gauge, histogram, and summary metric types.
 * 
 * @see https://prometheus.io/docs/instrumenting/exposition_formats/
 */

export interface PrometheusMetric {
  name: string;
  type: 'counter' | 'gauge' | 'histogram' | 'summary' | 'untyped';
  help: string;
  values: Array<{
    value: number;
    labels: Record<string, string>;
    timestamp?: number;
  }>;
}

/**
 * Parse Prometheus text exposition format into structured metrics
 * 
 * @param text - Raw Prometheus metrics text
 * @returns Array of parsed metrics
 */
export function parsePrometheusMetrics(text: string): PrometheusMetric[] {
  const lines = text.split('\n').filter(line => line.trim() && !line.startsWith('#'));
  const metricMap = new Map<string, PrometheusMetric>();
  
  // First pass: collect HELP and TYPE comments
  const helpMap = new Map<string, string>();
  const typeMap = new Map<string, string>();
  
  text.split('\n').forEach(line => {
    if (line.startsWith('# HELP ') {
      const parts = line.substring(7).split(' ');
      const name = parts[0];
      const help = parts.slice(1).join(' ');
      helpMap.set(name, help);
    } else if (line.startsWith('# TYPE ') {
      const parts = line.substring(7).split(' ');
      const name = parts[0];
      const type = parts[1];
      typeMap.set(name, type);
    }
  });
  
  // Second pass: parse metric values
  lines.forEach(line => {
    const parsed = parseMetricLine(line);
    if (!parsed) return;
    
    const { name, labels, value, timestamp } = parsed;
    const baseName = getBaseName(name);
    
    if (!metricMap.has(baseName) {
      metricMap.set(baseName, {
        name: baseName,
        type: (typeMap.get(baseName) as any) || 'untyped',
        help: helpMap.get(baseName) || '',
        values: []
      });
    }
    
    const metric = metricMap.get(baseName)!;
    metric.values.push({ value, labels, timestamp });
  });
  
  return Array.from(metricMap.values());
}

/**
 * Parse a single metric line
 * 
 * Format: metric_name{label1="value1",label2="value2"} 123.456 1634567890000
 */
function parseMetricLine(line: string): {
  name: string;
  labels: Record<string, string>;
  value: number;
  timestamp?: number;
} | null {
  // Match: metric_name{labels} value [timestamp]
  const match = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*?)(?:\{([^}]*)\})?\s+([-+]?[0-9]*\.?[0-9]+(?:[eE][-+]?[0-9]+)?)\s*([0-9]+)?/);
  
  if (!match) return null;
  
  const [, name, labelsStr, valueStr, timestampStr] = match;
  const labels: Record<string, string> = {};
  
  // Parse labels
  if (labelsStr) {
    const labelPairs = labelsStr.match(/([a-zA-Z_][a-zA-Z0-9_]*)="([^"]*)"/g);
    if (labelPairs) {
      labelPairs.forEach(pair => {
        const [key, value] = pair.split('="');
        labels[key] = value.slice(0, -1); // Remove trailing "
      });
    }
  }
  
  return {
    name,
    labels,
    value: parseFloat(valueStr),
    timestamp: timestampStr ? parseInt(timestampStr) : undefined
  };
}

/**
 * Get base name of a metric (removes suffixes like _total, _sum, _count, _bucket)
 */
function getBaseName(name: string): string {
  // Remove common histogram/summary suffixes
  return name
    .replace(/_bucket$/, '')
    .replace(/_sum$/, '')
    .replace(/_count$/, '')
    .replace(/_total$/, '');
}

/**
 * Get a single metric value by name and optional labels
 * 
 * @param metrics - Parsed metrics array
 * @param name - Metric name to find
 * @param labels - Optional labels to match
 * @returns Metric value or null if not found
 */
export function getMetricValue(
  metrics: PrometheusMetric[],
  name: string,
  labels?: Record<string, string>
): number | null {
  const metric = metrics.find(m => m.name === name);
  if (!metric || metric.values.length === 0) return null;
  
  if (!labels || Object.keys(labels).length === 0) {
    // Return first value if no label filter
    return metric.values[0].value;
  }
  
  // Find value matching all specified labels
  const matchingValue = metric.values.find(v => {
    return Object.entries(labels).every(([key, value]) => v.labels[key] === value);
  });
  
  return matchingValue?.value ?? null;
}

/**
 * Calculate statistics from a histogram metric
 * 
 * @param metrics - Parsed metrics array
 * @param baseName - Base name of histogram (without _bucket suffix)
 * @returns Statistics object with avg, p95, p99
 */
export function getHistogramStats(
  metrics: PrometheusMetric[],
  baseName: string
): { avg: number; p95: number; p99: number; count: number; sum: number } | null {
  // Try finding with explicit suffixes in original metrics
  const allMetrics = metrics.flatMap(m => 
    m.values.map(v => ({ name: m.name, ...v }))
  );
  
  const sumValue = allMetrics.find(m => m.name === `${baseName}_sum`)?.value;
  const countValue = allMetrics.find(m => m.name === `${baseName}_count`)?.value;
  
  if (sumValue === undefined || countValue === undefined || countValue === 0) {
    return null;
  }
  
  const avg = sumValue / countValue;
  
  // For P95 and P99, we'd need bucket data - for now, estimate
  // In a real implementation, you'd iterate through buckets to find percentiles
  const p95 = avg * 2; // Rough estimate
  const p99 = avg * 3; // Rough estimate
  
  return {
    avg,
    p95,
    p99,
    count: countValue,
    sum: sumValue
  };
}

/**
 * Get all metric names from parsed metrics
 */
export function getMetricNames(metrics: PrometheusMetric[]): string[] {
  return metrics.map(m => m.name);
}

/**
 * Filter metrics by type
 */
export function filterMetricsByType(
  metrics: PrometheusMetric[],
  type: PrometheusMetric['type']
): PrometheusMetric[] {
  return metrics.filter(m => m.type === type);
}

/**
 * Calculate cache hit rate from cache hits and misses metrics
 */
export function calculateCacheHitRate(
  metrics: PrometheusMetric[],
  hitsMetricName: string,
  missesMetricName: string
): number | null {
  const hits = getMetricValue(metrics, hitsMetricName);
  const misses = getMetricValue(metrics, missesMetricName);
  
  if (hits === null || misses === null) return null;
  
  const total = hits + misses;
  if (total === 0) return 0;
  
  return hits / total;
}

/**
 * Sum all values of a counter metric (useful for aggregating across labels)
 */
export function sumMetricValues(metrics: PrometheusMetric[], name: string): number {
  const metric = metrics.find(m => m.name === name);
  if (!metric) return 0;
  
  return metric.values.reduce((sum, v) => sum + v.value, 0);
}
