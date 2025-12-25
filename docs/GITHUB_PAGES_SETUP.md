---
title: Github Pages Setup
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth API - GitHub Pages Documentation

GitHub Pages is now set up for your OpenAPI documentation!

## 📁 Files Created

- `docs/index.html` - Landing page with links to all documentation
- `docs/swagger.html` - Swagger UI interface
- `docs/redoc.html` - ReDoc interface
- `docs/openapi.yaml` - Your OpenAPI specification

## 🚀 How to Enable GitHub Pages

1. **Push these files to GitHub:**
   ```bash
   git add docs/index.html docs/swagger.html docs/redoc.html docs/openapi.yaml
   git commit -m "Add GitHub Pages API documentation"
   git push origin main
   ```

2. **Enable GitHub Pages in your repository:**
   - Go to: https://github.com/mauriciomferz/Gauth_go/settings/pages
   - Under "Build and deployment":
     - **Source:** Deploy from a branch
     - **Branch:** main
     - **Folder:** /docs
   - Click **Save**

3. **Wait 1-2 minutes** for GitHub to build and deploy

## 🌐 Your Documentation URLs

Once enabled, your documentation will be available at:

- **Landing Page:** https://mauriciomferz.github.io/Gauth_go/
- **Swagger UI:** https://mauriciomferz.github.io/Gauth_go/swagger.html
- **ReDoc:** https://mauriciomferz.github.io/Gauth_go/redoc.html
- **OpenAPI Spec:** https://mauriciomferz.github.io/Gauth_go/openapi.yaml

## 🔄 Updating Documentation

Whenever you update your OpenAPI spec:

```bash
# Copy updated spec to docs
cp api/openapi/gauth.yaml docs/openapi.yaml

# Commit and push
git add docs/openapi.yaml
git commit -m "Update API documentation"
git push origin main
```

GitHub Pages will automatically rebuild in 1-2 minutes.

## ✨ Features

- **Beautiful landing page** with cards for each documentation type
- **Swagger UI** for interactive API testing
- **ReDoc** for clean, readable documentation
- **Direct YAML download** for SDK generation
- **Mobile responsive** design
- **No server required** - fully static

## 📝 Local Preview

You can preview the GitHub Pages locally:

```bash
# Open in browser
open docs/index.html
```

## 🔗 Share Your API

Once live, share this link with developers:
```
https://mauriciomferz.github.io/Gauth_go/
```

They can:
- Browse documentation via Swagger UI or ReDoc
- Download the OpenAPI spec
- Generate client SDKs using the spec URL
- Import into Postman, Insomnia, etc.

---

**Next Step:** Push to GitHub and enable Pages in repository settings!
