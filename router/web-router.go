package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/QuantumNous/ai-bridge/common"
	"github.com/QuantumNous/ai-bridge/controller"
	"github.com/QuantumNous/ai-bridge/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func SetWebRouter(router *gin.Engine, buildFS embed.FS, indexPage []byte) {
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(static.Serve("/", common.EmbedFolder(buildFS, "web/dist")))
	// Serve uploaded files (logos, etc.) from web/public/uploads
	router.Static("/uploads", "web/public/uploads")
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") ||
			strings.HasPrefix(c.Request.RequestURI, "/api") ||
			strings.HasPrefix(c.Request.RequestURI, "/assets") ||
			strings.HasPrefix(c.Request.RequestURI, "/metrics") ||
			strings.HasPrefix(c.Request.RequestURI, "/uploads") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPage)
	})
}
