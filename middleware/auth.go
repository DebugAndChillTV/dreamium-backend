package middleware

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	supa "github.com/supabase-community/supabase-go"
)

var supabaseClient *supa.Client

// GetSupabaseClient returns the initialized Supabase client
func GetSupabaseClient() *supa.Client {
	return supabaseClient
}

// Initialize Supabase client
func InitSupabase() {
	_ = godotenv.Load()

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_ADMIN_KEY")

	if supabaseURL == "" || supabaseKey == "" {
		log.Fatal("Missing Supabase credentials in .env")
	}

	var err error
	supabaseClient, err = supa.NewClient(supabaseURL, supabaseKey, nil)
	if err != nil {
		log.Fatalf("Failed to initialize Supabase client: %v", err)
	}
}

// Middleware to authenticate requests using Supabase JWT
func SupabaseAuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing Authorization Header"})
	}

	fmt.Println("authHeader", authHeader)

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid Authorization Format"})
	}

	fmt.Println("token", token)

	// Verify Supabase Auth Token
	client := supabaseClient.Auth.WithToken(token)
	user, err := client.GetUser()

	fmt.Println("user", user)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid Supabase Token"})
	}

	// Attach user ID to locals
	c.Locals("userID", user.ID)
	return c.Next()
}
