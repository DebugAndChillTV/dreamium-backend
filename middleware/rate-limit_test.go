package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiterMiddleware(t *testing.T) {
	rateLimiter.Flush()
	t.Cleanup(func() { rateLimiter.Flush() })

	app := fiber.New()
	app.Use(RateLimiterMiddleware())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)
		
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("Request %d failed: expected 200, got %d", i+1, resp.StatusCode)
		}
	}

	// The 11th request should be blocked
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode, "11th request should return 429")
}
