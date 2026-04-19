package routes

import (
	"dreamium-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	// Initialize OpenAI client first
	InitOpenAI()
	InitEncryption()
	// Protected routes (require authentication)
	api := app.Group("/api")
	api.Use(middleware.SupabaseAuthMiddleware)
	api.Post("/isDreamInput", IsDreamInput)
	api.Post("/generateDreamAnalysis", GenerateDreamAnalysis)
	api.Post("/generateSymbolicInterpretation", GenerateSymbolicInterpretation)
	api.Post("/generatePsychologistInterpretation", GeneratePsychologistInterpretation)
}
