---
title: Web Ui Usage Guide
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AgentAuth Web UI Usage Guide

## ✅ Status: All Pages Operational

All web pages are loading correctly and serving proper resources:

- **protocol-flow.html**: ✅ 9,993 bytes, HTTP 200
- **protocol-navigator.js**: ✅ 19,124 bytes, HTTP 200  
- **poa-visualization.html**: ✅ Fully loaded
- **poa-viz.js**: ✅ 22,778 bytes, HTTP 200
- **CSS files**: ✅ All accessible via `/static/css/:file`

## 🎯 How to Use the Pages

### 1. Protocol Flow Navigator (`/protocol-flow.html`)

**Purpose**: Interactive visualization of AgentAuth AAP-001 protocol flow

**How to Use**:
1. Navigate to `http://localhost:8080/protocol-flow.html`
2. The page loads with **Demo Controls** at the top
3. **Click the buttons** to interact:
   - `▶ Start Subscription` - Begins subscription flow
   - `🔍 Run Matching` - Executes PoA matching
   - `🎯 Request Authorization` - Demonstrates subset request
   - `🛡️ Enforce Policy` - Shows PEP enforcement
   - `✓ Verify Token` - Token verification process
   - `📊 Audit & Report` - Audit logging demo
   - `🎉 Complete Full Flow` - Runs entire protocol flow automatically
   - `⚠️ Simulate Error` - Shows error handling

**What You Should See**:
- After clicking buttons, the protocol navigator below updates
- Steps expand/collapse showing substeps
- Progress indicators change (pending → active → complete)
- Visual feedback with colors and animations

### 2. PoA Visualization (`/poa-visualization.html`)

**Purpose**: 3D visualization of Power of Attorney relationships using Three.js

**How to Use**:
1. Navigate to `http://localhost:8080/poa-visualization.html`
2. Use the **Control Panel** on the right:
   - **Visualization Mode**: Switch between "PoA Graph" and "Protocol Steps"
   - **Graph Type**: Select Demo/Simple/Multi-Party
   - **Load Button**: Click to render the visualization
   - **Rotate Toggle**: Enable/disable auto-rotation
   - **Reset View**: Return camera to default position

**What You Should See**:
- 3D canvas with grid background
- Nodes and edges in 3D space
- Interactive camera controls (click and drag to rotate)
- Statistics panel showing node/edge counts
- Scan line effect for aesthetic appeal

### 3. Comprehensive Demo (`/demo.html`)

**Purpose**: Complete AgentAuth demonstration page

**How to Use**:
1. Navigate to `http://localhost:8080/demo.html`
2. Page loads with all demo content
3. Interact with various components and examples

## 🔧 Development Mode

Server is running with:
- `AGENTAUTH_DEV_INDEX=1` - Enables disk-based HTML serving (hot reload)
- `AGENTAUTH_DEV_MODULES=1` - Enables disk-based module serving (hot reload)

**Benefits**:
- Edit HTML/CSS/JS files and refresh browser to see changes
- No need to rebuild or restart server
- Faster iteration during development

## 🐛 Troubleshooting

### "Page does nothing" or "Nothing happens"

**Diagnosis**: Page loaded successfully but appears static

**Solutions**:
1. **Click the interactive buttons** - Pages are intentionally non-animated until you interact
2. **Check browser console** (F12 → Console tab):
   - Look for JavaScript errors
   - Verify Three.js loaded from CDN
   - Check module imports succeeded
3. **Verify network requests** (F12 → Network tab):
   - All resources should show HTTP 200
   - CSS files loaded from `/static/css/`
   - JS modules loaded from `/static/js/modules/`

### Still Not Working?

Run these diagnostic commands:

```bash
# Check if server is running
curl http://localhost:8080/ready

# Test HTML page
curl -I http://localhost:8080/protocol-flow.html

# Test CSS
curl -I http://localhost:8080/static/css/protocol-navigator.css

# Test JS module
curl -I http://localhost:8080/static/js/modules/protocol-navigator.js

# Check server logs
tail -30 agentauth-web.log
```

All should return HTTP 200.

## 📊 Server Logs Confirmation

Latest logs show successful serving:
```
[debug] serving disk protocol-flow.html (9993 bytes)
[GIN] 2025/11/07 - 13:55:39 | 200 | 775.958µs | ::1 | GET "/protocol-flow.html"

[debug] served module protocol-navigator.js (19124 bytes)
[GIN] 2025/11/07 - 13:55:44 | 200 | 520.5µs | ::1 | GET "/static/js/modules/protocol-navigator.js"

[debug] served module poa-viz.js (22778 bytes)
[GIN] 2025/11/07 - 13:55:52 | 200 | 395.875µs | ::1 | GET "/static/js/modules/poa-viz.js"
```

## ✨ Next Steps

All Item 2 and Item 3 UIs are fully operational. Ready to proceed with:

- **Item 8**: Admin Cockpit Application
  - Integrate all UIs (protocol flow, PoA viz, G-Agent metrics)
  - Add operational dashboards
  - Create unified admin interface

## 📝 Notes

- Pages use ES6 modules (modern JavaScript)
- Three.js loaded from CDN (no local dependency)
- All pages are interactive - **user action required** to see animations
- Dev mode provides instant feedback for development

---

**Server**: http://localhost:8080  
**Status**: ✅ All systems operational  
**Last Updated**: November 7, 2025
