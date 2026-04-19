//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

func main() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Fatal("SENTRY_DSN environment variable is required")
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      "dev",
		Debug:            true,
		AttachStacktrace: true,
		SendDefaultPII:   false,
	})
	if err != nil {
		log.Fatalf("Sentry init failed: %v", err)
	}
	defer sentry.Flush(time.Second * 5)

	// Send test error
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("test", "true")
		scope.SetTag("source", "test_sentry.go")
		scope.SetLevel(sentry.LevelError)

		sentry.CaptureException(fmt.Errorf("this is a test error from Sentry integration"))
	})

	fmt.Println("Test error sent to Sentry successfully!")
	fmt.Println("Check your Sentry dashboard to verify.")
}
