package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "gh-pin-repo-go/docs"
)

// SetupRouter initializes the Gin engine and configures routes
func SetupRouter(handler *Handler) *gin.Engine {
	r := gin.New()

	// Use Recovery middleware
	r.Use(gin.Recovery())

	// Highly optimized CORS configuration
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Swagger documentation route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", handler.HealthCheck)

	// API group
	v1 := r.Group("/api/v1")
	{
		v1.GET("/pinned/:username", handler.GetPinnedRepos)
		v1.GET("/health", handler.HealthCheck)
	}

	// Redirect root to swagger UI, or handle legacy ?username= query
	r.GET("/", func(c *gin.Context) {
		username := c.Query("username")
		if username != "" {
			c.Params = append(c.Params, gin.Param{Key: "username", Value: username})
			// default to raw=true for legacy /?username= compatibility
			if c.Query("raw") == "" {
				q := c.Request.URL.Query()
				q.Set("raw", "true")
				c.Request.URL.RawQuery = q.Encode()
			}
			handler.GetPinnedRepos(c)
			return
		}
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	return r
}
