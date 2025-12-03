# E2E Testing with Playwright

## Overview

This directory contains end-to-end tests for the GAuth React UI using Playwright.

## Test Structure

```
e2e/
├── overview.spec.ts       # Overview page tests (10 test cases)
├── tokens.spec.ts         # Token management tests  
├── navigation.spec.ts     # Navigation & common element tests
└── all-pages.spec.ts      # Tests for PIP, PoA, E2E Testing, Metrics pages
```

## Running Tests

### Install Playwright browsers (first time only)
```bash
npx playwright install
```

### Run all tests
```bash
npm test
```

### Run tests in UI mode (interactive)
```bash
npm run test:ui
```

### Run tests in headed mode (watch browsers)
```bash
npm run test:headed
```

### Run specific browser
```bash
npm run test:chromium
npm run test:firefox
npm run test:webkit
```

### View test report
```bash
npm run test:report
```

### Generate tests interactively
```bash
npm run test:codegen
```

## Test Coverage

### Overview Page (overview.spec.ts)
- ✅ Main heading display
- ✅ RFC compliance information
- ✅ Stat cards (4 cards)
- ✅ Backend status indicator
- ✅ RFC compliance section
- ✅ System components
- ✅ Quick start section
- ✅ Navigation links
- ✅ Responsive design (mobile/tablet/desktop)
- ✅ Theme toggle

### Tokens Page (tokens.spec.ts)
- ✅ Page heading
- ✅ Create token form
- ✅ Validate token form
- ✅ Recent tokens section
- ✅ Form submission
- ✅ Input validation
- ✅ Token list display

### Navigation (navigation.spec.ts)
- ✅ Navigate all 8 pages
- ✅ Consistent header
- ✅ Footer presence
- ✅ No console errors
- ✅ Graceful API error handling

### Other Pages (all-pages.spec.ts)
- ✅ PIP authorization form & cache stats
- ✅ PoA creation form & RFC-0115 compliance
- ✅ E2E Testing controls & coverage
- ✅ Metrics dashboard & stat cards

## Configuration

See `playwright.config.ts` for detailed configuration including:
- Base URL: http://localhost:3001
- Browsers: Chromium, Firefox, WebKit, Mobile Chrome, Mobile Safari
- Reporters: HTML, JSON, List
- Traces: On first retry
- Screenshots: Only on failure
- Video: Retain on failure

## CI/CD Integration

Tests are configured to run in CI environments with:
- Fail on `.only` tests
- 2 retries on failure
- Single worker (no parallel execution)
- Automatic dev server startup

## Best Practices

1. **Before Each Test**: Navigate to the page under test
2. **Assertions**: Use `toBeVisible()` with appropriate timeouts
3. **Waiting**: Use `page.waitForTimeout()` sparingly, prefer `waitForSelector()`
4. **Selectors**: Prefer text selectors and accessible attributes
5. **Error Handling**: Handle expected API errors gracefully

## Debugging

```bash
# Debug specific test
npx playwright test overview.spec.ts --debug

# Run test with traces
npx playwright test --trace on

# View trace
npx playwright show-trace trace.zip
```

## Requirements

- Node.js 18+
- Frontend dev server running on http://localhost:3001
- Backend API server running on http://localhost:8080 (optional, tests handle failures)

## Troubleshooting

### Tests timing out
- Ensure dev server is running: `npm run dev`
- Increase timeout in `playwright.config.ts`

### Browser not launching
- Run: `npx playwright install`
- Check for system dependencies: `npx playwright install-deps`

### API connection errors
- Tests are designed to handle API failures gracefully
- Backend connection errors are filtered from critical errors

## Test Metrics

- **Total Test Files**: 4
- **Estimated Test Cases**: 25+
- **Coverage**: All 8 pages
- **Browsers**: 5 (Desktop Chrome/Firefox/Safari, Mobile Chrome/Safari)
- **Execution Time**: ~2-5 minutes (full suite)
