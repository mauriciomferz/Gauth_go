// Package docs provides HTTP handlers for serving API documentation (Swagger UI and ReDoc)
package docs

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed swagger-ui.html
var swaggerUIHTML string

//go:embed redoc.html
var redocHTML string

// SwaggerUIHandler serves the Swagger UI interface
func SwaggerUIHandler(c *gin.Context) {
	// Get the base URL for the API spec
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	specURL := scheme + "://" + host + "/openapi.yaml"

	// Get the CSP nonce from the context
	nonce, _ := c.Get("csp-nonce")
	nonceStr, _ := nonce.(string)

	// Replace placeholders in the HTML
	html := strings.ReplaceAll(swaggerUIHTML, "{{SPEC_URL}}", specURL)
	html = strings.ReplaceAll(html, "{{NONCE}}", nonceStr)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.String(http.StatusOK, html)
}

// ReDocHandler serves the ReDoc interface
func ReDocHandler(c *gin.Context) {
	// Get the base URL for the API spec
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	specURL := scheme + "://" + host + "/openapi.yaml"

	// Get the CSP nonce from the context
	nonce, _ := c.Get("csp-nonce")
	nonceStr, _ := nonce.(string)

	// Replace placeholders in the HTML
	html := strings.ReplaceAll(redocHTML, "{{SPEC_URL}}", specURL)
	html = strings.ReplaceAll(html, "{{NONCE}}", nonceStr)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.String(http.StatusOK, html)
}

// OpenAPISpecHandler serves the OpenAPI specification file
func OpenAPISpecHandler(c *gin.Context) {
	// In production, this would read from the embedded file
	// For now, serve from the file system
	c.File("./docs/openapi/agentauth-api.yaml")
}

// DocsLandingHandler serves a landing page with links to both documentation formats
func DocsLandingHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AgentAuth API Documentation</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 800px;
            width: 100%;
            padding: 60px 40px;
            text-align: center;
        }
        h1 {
            color: #2d3748;
            font-size: 2.5rem;
            margin-bottom: 10px;
        }
        .subtitle {
            color: #718096;
            font-size: 1.1rem;
            margin-bottom: 40px;
        }
        .version-badge {
            display: inline-block;
            background: #667eea;
            color: white;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 0.9rem;
            margin-bottom: 30px;
        }
        .cards {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        .card {
            background: #f7fafc;
            border: 2px solid #e2e8f0;
            border-radius: 12px;
            padding: 30px;
            text-decoration: none;
            color: inherit;
            transition: all 0.3s ease;
        }
        .card:hover {
            transform: translateY(-5px);
            box-shadow: 0 10px 30px rgba(102, 126, 234, 0.3);
            border-color: #667eea;
        }
        .card-icon {
            font-size: 3rem;
            margin-bottom: 15px;
        }
        .card-title {
            font-size: 1.5rem;
            color: #2d3748;
            margin-bottom: 10px;
            font-weight: 600;
        }
        .card-description {
            color: #718096;
            font-size: 0.95rem;
            line-height: 1.5;
        }
        .features {
            text-align: left;
            background: #f7fafc;
            border-radius: 12px;
            padding: 30px;
            margin-top: 30px;
        }
        .features h2 {
            color: #2d3748;
            font-size: 1.5rem;
            margin-bottom: 20px;
        }
        .features ul {
            list-style: none;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
        }
        .features li {
            color: #4a5568;
            padding-left: 25px;
            position: relative;
        }
        .features li:before {
            content: "✓";
            position: absolute;
            left: 0;
            color: #667eea;
            font-weight: bold;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #e2e8f0;
            color: #718096;
            font-size: 0.9rem;
        }
        .footer a {
            color: #667eea;
            text-decoration: none;
        }
        .footer a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 AgentAuth API Documentation</h1>
        <p class="subtitle">OAuth 2.0 Authorization Server with RFC-0111 Compliance</p>
        <div class="version-badge">v1.0.0-beta</div>
        
        <div class="cards">
            <a href="/api/docs/swagger" class="card">
                <div class="card-icon">📘</div>
                <div class="card-title">Swagger UI</div>
                <div class="card-description">
                    Interactive API documentation with try-it-out functionality. 
                    Test endpoints directly from your browser.
                </div>
            </a>
            
            <a href="/api/docs/redoc" class="card">
                <div class="card-icon">📗</div>
                <div class="card-title">ReDoc</div>
                <div class="card-description">
                    Clean, responsive documentation with advanced search. 
                    Perfect for reading and reference.
                </div>
            </a>
        </div>

        <div class="features">
            <h2>Key Features</h2>
            <ul>
                <li>RFC-0111 Subscription Flow</li>
                <li>Power of Attorney Management</li>
                <li>Policy-Based Authorization</li>
                <li>Identity Verification (PVP)</li>
                <li>Commercial Registry Integration</li>
                <li>Token Lifecycle Management</li>
                <li>Prometheus Metrics</li>
                <li>Comprehensive Audit Logging</li>
                <li>Rate Limiting & Security</li>
                <li>Key Rotation Support</li>
            </ul>
        </div>

        <div class="footer">
            <p>
                <a href="/api/docs/openapi.yaml" target="_blank">📄 Download OpenAPI Spec</a> | 
                <a href="https://github.com/mauriciomferz/AgentAuth" target="_blank">GitHub</a> | 
                <a href="/api/v1/beta/health">Health Check</a>
            </p>
            <p style="margin-top: 10px;">
                Made with ❤️ by the AgentAuth Team
            </p>
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.String(http.StatusOK, html)
}
