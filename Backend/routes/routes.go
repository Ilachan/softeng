package routes

import (
	// Import your API layer (Ensure module name matches go.mod)
	"my-course-backend/api" 

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes the Gin engine and defines all routes
func SetupRouter() *gin.Engine {
	// Initialize Gin with default middleware (Logger and Recovery)
	r := gin.Default()

	// 1. Configure CORS (Cross-Origin Resource Sharing)
	// This logic was moved here from main.go to keep the entry point clean.
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true // Warning: Allow all origins for development; restrict in production.
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	
	// Apply CORS middleware globally
	r.Use(cors.New(config))

	// 2. Register Route Groups
	
	// Auth Route Group
	// Prefix: /auth
	authRoutes := r.Group("/auth")
	{
		// POST /auth/register
		authRoutes.POST("/register", api.Register)
		// POST /auth/login
		authRoutes.POST("/login", api.Login)
	}

	// User Route Group
	// Prefix: /users
	userRoutes := r.Group("/users")
	{
		// DELETE /users/:id (e.g., /users/1)
		userRoutes.DELETE("/:id", api.DeleteUser)
	}

	// Note: Future Course Route Group can be added here...
	// courseRoutes := r.Group("/courses") { ... }

	return r
}