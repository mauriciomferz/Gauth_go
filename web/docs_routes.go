package web

import (
	"github.com/mauriciomferz/AgentAuth/web/handlers/docs"
)

// RegisterAPIDocumentation registers API documentation endpoints (Swagger UI, ReDoc, OpenAPI spec)
func (s *BetaServer) RegisterAPIDocumentation() {
	docsGroup := s.router.Group("/api/docs")
	{
		// Landing page with links to Swagger UI and ReDoc
		docsGroup.GET("", docs.DocsLandingHandler)
		docsGroup.GET("/", docs.DocsLandingHandler)

		// Swagger UI
		docsGroup.GET("/swagger", docs.SwaggerUIHandler)

		// ReDoc
		docsGroup.GET("/redoc", docs.ReDocHandler)

		// OpenAPI specification
		docsGroup.GET("/openapi.yaml", docs.OpenAPISpecHandler)
		docsGroup.GET("/openapi.yml", docs.OpenAPISpecHandler)
		docsGroup.GET("/spec", docs.OpenAPISpecHandler)
	}
}
