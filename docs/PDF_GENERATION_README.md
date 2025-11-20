# GAuth API Documentation - PDF/HTML Generation

## ✅ HTML Files Generated Successfully!

Three HTML documentation files have been created in the `docs/` directory:

- **API_EXAMPLES.html** (120 KB) - Code examples and usage patterns
- **API_REFERENCE.html** (189 KB) - Complete API endpoint reference  
- **API_VERSIONING.html** (33 KB) - API versioning guide

## 📄 Converting HTML to PDF

### Option 1: Using Your Browser (Recommended - Easy)

1. Open the HTML file in any browser:
   ```bash
   open docs/API_REFERENCE.html
   # or
   open docs/API_EXAMPLES.html
   # or
   open docs/API_VERSIONING.html
   ```

2. Press `Cmd + P` (Print)

3. In the print dialog:
   - Destination: **Save as PDF**
   - Layout: Portrait
   - Margins: Default
   - Options: ✓ Background graphics

4. Click **Save** and choose your filename

### Option 2: Using Command Line (wkhtmltopdf)

```bash
# Install wkhtmltopdf
brew install wkhtmltopdf

# Convert to PDF
cd docs
wkhtmltopdf API_EXAMPLES.html API_EXAMPLES.pdf
wkhtmltopdf API_REFERENCE.html API_REFERENCE.pdf
wkhtmltopdf API_VERSIONING.html API_VERSIONING.pdf
```

### Option 3: Installing LaTeX for Direct PDF Generation

If you want to generate PDFs directly from Markdown without the HTML step:

```bash
# Install BasicTeX (small, ~100MB)
brew install --cask basictex

# Add LaTeX to PATH
export PATH="/Library/TeX/texbin:$PATH"

# Install required packages
sudo tlmgr update --self
sudo tlmgr install framed fvextra footnotebackref pagecolor mdframed needspace \
  sourcecodepro sourcesanspro fontawesome5 tcolorbox environ

# Then run the script without --html flag
./generate_api_docs_pdf.sh
```

## 🚀 Quick Access

Open the HTML files in your browser:

```bash
# Open all three files
open docs/API_EXAMPLES.html docs/API_REFERENCE.html docs/API_VERSIONING.html

# Or individually
open docs/API_REFERENCE.html
```

## 📋 Script Usage

```bash
# Generate HTML files (no LaTeX required)
./generate_api_docs_pdf.sh --html

# Generate PDFs directly (requires LaTeX)
./generate_api_docs_pdf.sh
```

## 🎨 Styling

The HTML files use GitHub Markdown CSS for professional formatting:
- Clean, readable typography
- Syntax highlighting for code blocks
- Responsive design
- Print-friendly layout

## 📊 File Sizes

- **API_EXAMPLES.html**: 120 KB
- **API_REFERENCE.html**: 189 KB (largest - complete endpoint reference)
- **API_VERSIONING.html**: 33 KB

## ✨ Features

- ✅ Table of Contents
- ✅ Syntax-highlighted code blocks
- ✅ Numbered sections
- ✅ Professional GitHub styling
- ✅ Print-optimized layout
- ✅ Clickable internal links

## 🔗 Alternative: View Online

You can also view these files directly on GitHub or use the Swagger UI:

```
http://localhost:8080/api/docs
```

---

**Generated:** November 17, 2025  
**Script:** `generate_api_docs_pdf.sh`  
**Format:** HTML (convertible to PDF via browser print)
