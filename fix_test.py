import re

# Read the file
with open('examples/resilient/main_test.go', 'r') as f:
    content = f.read()

# Replace the specific problematic lines in MetricsCollection test
# Replace the metrics reset and breaker reset with fresh service creation
content = re.sub(
    r'(\s+)// Reset metrics and circuit breaker state\n\s+service\.metrics = monitoring\.NewMetricsCollector\(\)\n\s+service\.breaker\.Reset\(\)',
    r'\1// Create a fresh service instance to avoid race conditions with shared state\n\1freshService := NewResilientService(auth)',
    content
)

# Replace service.ProcessRequest with freshService.ProcessRequest in the transactions loop
content = re.sub(
    r'(\s+for _, tc := range transactions \{\n\s+)_ = service\.ProcessRequest\(tc\.tx, tc\.token\)',
    r'\1_ = freshService.ProcessRequest(tc.tx, tc.token)',
    content
)

# Replace service.metrics with freshService.metrics in the MetricsCollection test
content = re.sub(
    r'(\s+metrics := )service\.metrics\.GetAllMetrics\(\)',
    r'\1freshService.metrics.GetAllMetrics()',
    content
)

# Write the fixed content
with open('examples/resilient/main_test.go', 'w') as f:
    f.write(content)

print("Fixed MetricsCollection test race condition")
