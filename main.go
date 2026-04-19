package main

import (
	"dreamium-backend/middleware"
	"dreamium-backend/routes"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables.")
	}

	// Initialize Supabase
	middleware.InitSupabase()

	// Initialize Fiber app
	app := fiber.New()

	// Apply rate limiter globally
	app.Use(middleware.RateLimiterMiddleware())

	// Register routes (this will also initialize OpenAI)
	routes.SetupRoutes(app)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	log.Printf("Dreamium backend starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
