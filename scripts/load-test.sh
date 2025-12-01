#!/bin/bash
# Simple load test for blue-green deployment

NAMESPACE="gauth-staging"
SERVICE="gauth-service"
REQUESTS=100
CONCURRENCY=10

echo "Running load test..."
echo "Target: http://$SERVICE.$NAMESPACE.svc.cluster.local/api/v1/beta/health"
echo "Requests: $REQUESTS, Concurrency: $CONCURRENCY"
echo ""

kubectl run load-test --rm -i --tty \
  --image=curlimages/curl:latest \
  --restart=Never \
  -n $NAMESPACE \
  -- sh -c "
    SUCCESS=0
    FAILED=0
    TOTAL=0
    START=\$(date +%s)
    
    for i in \$(seq 1 $REQUESTS); do
      RESPONSE=\$(curl -s -w '%{http_code}' -o /dev/null http://$SERVICE/api/v1/beta/health)
      if [ \"\$RESPONSE\" = \"200\" ]; then
        SUCCESS=\$((SUCCESS + 1))
      else
        FAILED=\$((FAILED + 1))
      fi
      TOTAL=\$((TOTAL + 1))
      
      if [ \$((TOTAL % 10)) -eq 0 ]; then
        echo \"Progress: \$TOTAL/$REQUESTS\"
      fi
    done
    
    END=\$(date +%s)
    DURATION=\$((END - START))
    
    echo \"\"
    echo \"=== Load Test Results ===\"
    echo \"Total requests: \$TOTAL\"
    echo \"Successful: \$SUCCESS\"
    echo \"Failed: \$FAILED\"
    echo \"Duration: \${DURATION}s\"
    echo \"Requests/sec: \$((TOTAL / DURATION))\"
  "
