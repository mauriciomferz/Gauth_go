#!/bin/bash
# Check CI/CD Status Script
# Monitors the status of GitHub Actions workflows

echo "🔍 Checking CI/CD Workflow Status..."
echo "======================================"

echo ""
echo "📋 Latest Commits:"
git log --oneline -3

echo ""
echo "🌐 Repository Remotes:"
git remote -v

echo ""
echo "📁 Available Build Artifacts:"
ls -la gauth-* 2>/dev/null || echo "No build artifacts found"

echo ""
echo "🏗️ Local Build Test Results:"
echo "Main Demo Build:"
if go build -v -o test-gauth-demo ./cmd/demo 2>/dev/null; then
    echo "✅ Main demo builds successfully"
    rm -f test-gauth-demo
else
    echo "❌ Main demo build failed"
fi

echo ""
echo "Web Backend Build:"
if cd gauth-demo-app/web/backend; then
    if go build -v -o ../../../test-gauth-web-backend ./ 2>/dev/null; then
        echo "✅ Web backend builds successfully"
        cd ../../.. || exit
        rm -f test-gauth-web-backend
    else
        echo "❌ Web backend build failed"
        cd ../../.. || exit
    fi
else
    echo "❌ Cannot access web backend directory"
fi

echo ""
echo "🔧 CI/CD Workflow File Status:"
if [ -f ".github/workflows/ci-cd.yml" ]; then
    echo "✅ CI/CD workflow file exists"
    echo "📝 Build section preview:"
    grep -A 10 "Build binaries" .github/workflows/ci-cd.yml | head -15
else
    echo "❌ CI/CD workflow file not found"
fi

echo ""
echo "🎯 Next Steps:"
echo "1. Check GitHub Actions tab in each repository:"
echo "   - https://github.com/mauriciomferz/Gauth_go/actions"
echo "   - https://github.com/Gimel-Foundation/Gimel-App-0001/actions"
echo "   - https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/actions"
echo "2. Monitor workflow execution for successful builds"
echo "3. Verify all tests pass with updated build paths"