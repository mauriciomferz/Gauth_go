package patterns
package main

import (
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/examples/patterns"
)

func main() {
	fmt.Println("\nRunning Resilience Pattern Examples...")
	fmt.Println("=====================================")

	// Run Distributed Cache example
	fmt.Println("\n1. Distributed Cache Example")
	fmt.Println("----------------------------")
	patterns.RunDemo()

	// Add some separation between examples
	time.Sleep(2 * time.Second)

	// Run API Gateway example
	fmt.Println("\n2. API Gateway Example")
	fmt.Println("---------------------")
	patterns.RunAPIGatewayDemo()

	fmt.Println("\nAll examples completed!")
}