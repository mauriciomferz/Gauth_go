package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
)

func main() {
	// Create a rate limiter allowing 5 requests per second with a 2-second window
	config := &ratelimit.Config{
		RequestsPerSecond: 5,
		WindowSize:        2,
		BurstSize:         10, // Allow bursts up to 10 requests
	}
	limiter := ratelimit.NewLimiter(config)

	// Simulate multiple clients making requests concurrently
	var wg sync.WaitGroup
	ctx := context.Background()

	// Simulate client 1: Burst of requests
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientID := "client1"
		fmt.Printf("\nClient 1 - Burst test (should see some failures):\n")

		// Try to make 8 requests in quick succession
		for i := 1; i <= 8; i++ {
			err := limiter.Allow(ctx, clientID)
			if err != nil {
				fmt.Printf("Request %d: Rate limit exceeded\n", i)
			} else {
				fmt.Printf("Request %d: Allowed\n", i)
			}

			// Check remaining requests after each attempt
			remaining := limiter.GetRemainingRequests(clientID)
			fmt.Printf("Remaining requests: %d\n", remaining)
		}
	}()

	// Simulate client 2: Steady rate
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientID := "client2"
		fmt.Printf("\nClient 2 - Steady rate test:\n")

		// Make 5 requests with 500ms delay between them
		for i := 1; i <= 5; i++ {
			err := limiter.Allow(ctx, clientID)
			if err != nil {
				fmt.Printf("Request %d: Rate limit exceeded\n", i)
			} else {
				fmt.Printf("Request %d: Allowed\n", i)
			}

			remaining := limiter.GetRemainingRequests(clientID)
			fmt.Printf("Remaining requests: %d\n", remaining)

			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Simulate client 3: Reset and retry
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientID := "client3"
		fmt.Printf("\nClient 3 - Reset test:\n")

		// Make requests until we hit the limit
		for i := 1; ; i++ {
			err := limiter.Allow(ctx, clientID)
			if err != nil {
				fmt.Printf("Hit rate limit after %d requests\n", i-1)

				// Reset the window and try one more request
				limiter.Reset(clientID)
				fmt.Println("Reset window...")

				err = limiter.Allow(ctx, clientID)
				if err != nil {
					fmt.Println("Post-reset request: Rate limit exceeded")
				} else {
					fmt.Println("Post-reset request: Allowed")
				}
				break
			}
		}
	}()

	// Simulate client 4: Remove and retry
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientID := "client4"
		fmt.Printf("\nClient 4 - Remove test:\n")

		// Make requests until we hit the limit
		for i := 1; ; i++ {
			err := limiter.Allow(ctx, clientID)
			if err != nil {
				fmt.Printf("Hit rate limit after %d requests\n", i-1)

				// Remove the window and try one more request
				limiter.Remove(clientID)
				fmt.Println("Removed rate limit tracking...")

				err = limiter.Allow(ctx, clientID)
				if err != nil {
					fmt.Println("Post-remove request: Rate limit exceeded")
				} else {
					fmt.Println("Post-remove request: Allowed")
				}
				break
			}
		}
	}()

	// Wait for all clients to finish
	wg.Wait()

	// Demonstrate window sliding
	fmt.Printf("\nWindow sliding test:\n")
	clientID := "client5"

	// Fill up the window
	for i := 1; i <= config.RequestsPerSecond; i++ {
		limiter.Allow(ctx, clientID)
		fmt.Printf("Initial request %d: Allowed\n", i)
	}

	// Try one more request (should fail)
	if err := limiter.Allow(ctx, clientID); err != nil {
		fmt.Println("Extra request: Rate limit exceeded")
	}

	// Wait for window to slide
	fmt.Println("Waiting for window to slide...")
	time.Sleep(time.Duration(config.WindowSize) * time.Second)

	// Try again (should succeed)
	if err := limiter.Allow(ctx, clientID); err != nil {
		fmt.Println("Request after window slide: Rate limit exceeded")
	} else {
		fmt.Println("Request after window slide: Allowed")
	}
}
