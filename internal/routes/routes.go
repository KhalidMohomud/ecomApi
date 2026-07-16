// Package routes is the only place that maps URL paths to handler
// methods. Nothing about business logic or persistence lives here —
// just wiring.
package routes

import (
	"net/http"

	_ "github.com/KhalidMohomud/ecomApi/docs" // swag-generated spec; blank import runs its init(), registering the spec with gin-swagger
	"github.com/KhalidMohomud/ecomApi/internal/config"
	"github.com/KhalidMohomud/ecomApi/internal/handler"
	"github.com/KhalidMohomud/ecomApi/internal/middleware"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handlers bundles every feature handler the router wires up. As
// later steps add Products, Cart, Orders, etc., they get one new
// field here rather than SetupRouter growing a longer and longer
// parameter list.
type Handlers struct {
	Auth *handler.AuthHandler
	User *handler.UserHandler
}

// SetupRouter builds the fully wired Gin engine: global middleware,
// health check, Swagger UI, and every versioned API route.
//
// tokenManager is passed separately from Handlers because it's not a
// handler — it's what the Auth middleware itself needs to verify
// tokens on every protected route below.
func SetupRouter(cfg *config.Config, tokenManager *utils.TokenManager, h Handlers) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery(), middleware.RequestLogger())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "ok",
			"data":    gin.H{"env": cfg.App.Env},
		})
	})

	// Swagger UI, generated from the @-comments on main() and each
	// handler method. Regenerate with `swag init -g cmd/server/main.go -o docs`
	// any time an endpoint or its doc comment changes.
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
			auth.POST("/refresh", h.Auth.RefreshToken)
			auth.POST("/logout", h.Auth.Logout)
		}

		// Every route in this group requires a valid access token.
		// middleware.Auth populates the user ID/role that
		// UserHandler (and, later, any authenticated route) reads
		// via middleware.UserIDFromContext / RoleFromContext.
		protected := v1.Group("")
		protected.Use(middleware.Auth(tokenManager))
		{
			users := protected.Group("/users")
			{
				users.GET("/me", h.User.GetProfile)
				users.PUT("/me", h.User.UpdateProfile)
				users.PUT("/me/password", h.User.ChangePassword)
				users.DELETE("/me", h.User.DeleteAccount)
			}
		}
	}

	return router
}
