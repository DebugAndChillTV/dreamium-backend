package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/patrickmn/go-cache"
)

// Simple in-memory rate limiter
var rateLimiter = cache.New(1*time.Minute, 10*time.Minute)

// Rate Limiting Middleware
func RateLimiterMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		count, found := rateLimiter.Get(ip)

		if found && count.(int) >= 10 { // Limit: 10 requests per minute
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Too many requests"})
		}

		if found {
			rateLimiter.Set(ip, count.(int)+1, cache.DefaultExpiration)
		} else {
			rateLimiter.Set(ip, 1, cache.DefaultExpiration)
		}

		return c.Next()
	}
}
