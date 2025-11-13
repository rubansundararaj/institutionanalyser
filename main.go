package main

import (
	"fmt"
	"log"
	"os"

	"institutionanalyser/models"
	"institutionanalyser/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Institution Analyser API")

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		fmt.Println("Note: .env file not found, using environment variables only")
	}

	// Get database connection string
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		log.Fatal("DATABASE_URL environment variable is required. Please set it in your .env file or as an environment variable.")
	}

	// Initialize database
	db, err := models.InitDatabase(dbDSN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	fmt.Println("Database connection established successfully")

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Set Gin mode
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "release" // Default to release mode for production
	}
	gin.SetMode(ginMode)

	// Initialize router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "institution-analyser-api",
		})
	})

	routes.SetupRoutes(router, db)

	// Root endpoint

	// Start server
	fmt.Printf("Starting server on port %s...\n", port)
	fmt.Printf("API available at http://localhost:%s/api/v1\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
