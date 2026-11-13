package http

import (
	_ "embed"
	nethttp "net/http"

	"github.com/gin-gonic/gin"
)

// openAPIContentType is the media type of the OpenAPI document.
const openAPIContentType = "application/yaml; charset=utf-8"

// openAPISpec is the OpenAPI document describing every route of the API. It
// must be updated with the routes: tests fail when they diverge.
//
//go:embed openapi.yaml
var openAPISpec []byte

// serveOpenAPISpec handles GET /openapi.yaml.
func serveOpenAPISpec(c *gin.Context) {
	c.Data(nethttp.StatusOK, openAPIContentType, openAPISpec)
}
