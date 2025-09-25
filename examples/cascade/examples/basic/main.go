package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/examples/cascade/pkg/mesh"
)

// This example shows how to use individual resilience patterns
func main() {
	// Create services with specific resilience patterns
	paymentSvc := mesh.NewMicroservice(mesh.PaymentService, "Payment", nil)

	// Simulate increasing load
	for load := 0.0; load <= 1.0; load += 0.2 {
		paymentSvc.SetLoadFactor(load)

		fmt.Printf("\nTesting with load factor %.1f:\n", load)

		// Process multiple requests
		for i := 0; i < 10; i++ {
			ctx := context.Background()
			result, err := paymentSvc.ProcessRequest(ctx, nil)

			if err != nil {
				fmt.Printf("Request %d failed: %v\n", i+1, err)
			} else {
				fmt.Printf("Request %d succeeded: %v\n", i+1, result)
			}

			time.Sleep(100 * time.Millisecond)
		}
	}
}
