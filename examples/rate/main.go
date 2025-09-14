package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/rate"
)

func demoBasicUsage(limiter *rate.Limiter) {
	ctx := context.Background()
	id := "basic-demo"

	fmt.Println("\nBasic Usage Demo:")
	for i := 1; i <= 3; i++ {
		if err := limiter.Allow(ctx, id); err != nil {
			fmt.Printf("Request %d: Rate limit exceeded\n", i)
		} else {
			fmt.Printf("Request %d: Allowed\n", i)
		}
		remaining := limiter.GetRemainingRequests(id)
		fmt.Printf("Remaining requests: %d\n", remaining)
	}
}

func demoConcurrentClients(limiter *rate.Limiter) {
	ctx := context.Background()
	var wg sync.WaitGroup

	fmt.Println("\nConcurrent Clients Demo:")

	// Start 3 concurrent clients
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			id := fmt.Sprintf("client-%d", clientID)

			// Each client makes 3 requests
			for j := 1; j <= 3; j++ {
				if err := limiter.Allow(ctx, id); err != nil {
					fmt.Printf("Client %d, Request %d: Rate limit exceeded\n", clientID, j)
				} else {
					fmt.Printf("Client %d, Request %d: Allowed\n", clientID, j)
				}
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}
	wg.Wait()
}

func demoWindowSliding(limiter *rate.Limiter) {
	ctx := context.Background()
	id := "window-demo"

	fmt.Println("\nWindow Sliding Demo:")

	// Fill up the window
	for i := 1; i <= 5; i++ {
		limiter.Allow(ctx, id)
		fmt.Printf("Initial request %d: Allowed\n", i)
	}

	// Try one more (should fail)
	if err := limiter.Allow(ctx, id); err != nil {
		fmt.Println("Extra request: Rate limit exceeded")
	}

	// Wait for window to slide
	fmt.Println("Waiting for window to slide...")
	time.Sleep(2 * time.Second)

	// Try again (should work)
	if err := limiter.Allow(ctx, id); err != nil {
		fmt.Println("Request after slide: Rate limit exceeded")
	} else {
		fmt.Println("Request after slide: Allowed")
	}
}

func demoResetAndRetry(limiter *rate.Limiter) {
	ctx := context.Background()
	id := "reset-demo"

	fmt.Println("\nReset and Retry Demo:")

	// Fill up the window
	var i int
	for i = 1; ; i++ {
		if err := limiter.Allow(ctx, id); err != nil {
			fmt.Printf("Hit limit after %d requests\n", i-1)
			break
		}
	}

	// Reset and try again
	fmt.Println("Resetting window...")
	limiter.Reset(id)

	if err := limiter.Allow(ctx, id); err != nil {
		fmt.Println("Post-reset request: Rate limit exceeded")
	} else {
		fmt.Println("Post-reset request: Allowed")
	}
}

func main() {
	// Create a rate limiter
	config := &rate.Config{
		RequestsPerSecond: 5,  // 5 requests per second
		WindowSize:        2,  // Over a 2-second window
		BurstSize:         10, // Allow bursts up to 10
	}
	limiter := rate.NewLimiter(config)
	defer limiter.Close() // Ensure cleanup goroutine stops

	// Run demos
	demoBasicUsage(limiter)
	demoConcurrentClients(limiter)
	demoWindowSliding(limiter)
	demoResetAndRetry(limiter)
}
