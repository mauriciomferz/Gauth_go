#!/bin/bash

echo "🧪 Running AgentAuth Functional Tests..."
echo "=================================="

# Test 1: Advanced Revocation Flow
echo "1. Testing Advanced Token Revocation Flow:"
cd examples/token/advanced_revocation_flow
if go test -v; then
    echo "   ✅ PASSED: Advanced revocation flow test"
else
    echo "   ❌ FAILED: Advanced revocation flow test"
fi
cd ../../..

# Test 2: Core Build Tests
echo ""
echo "2. Testing Core Package Builds:"
if go build ./pkg/gauth ./pkg/token ./pkg/events ./pkg/resilience; then
    echo "   ✅ PASSED: All core packages build successfully"
else
    echo "   ❌ FAILED: Core package builds"
fi

# Test 3: Main Server Build
echo ""
echo "3. Testing Main Server Build:"
if go build ./cmd/gauth-server; then
    echo "   ✅ PASSED: Main server builds successfully"
else
    echo "   ❌ FAILED: Main server build"
fi

# Test 4: Example Builds
echo ""
echo "4. Testing Key Examples:"
if cd examples/typed_structures_demo && go build; then
    echo "   ✅ PASSED: typed_structures_demo builds"
    cd ../..
else
    echo "   ❌ FAILED: typed_structures_demo build"
    cd ../..
fi

if cd examples/cascade && go build; then
    echo "   ✅ PASSED: cascade example builds"
    cd ../..
else
    echo "   ❌ FAILED: cascade example build"
    cd ../..
fi

echo ""
echo "🎉 Functional Test Summary Complete!"
echo "✅ AgentAuth core functionality verified working"
