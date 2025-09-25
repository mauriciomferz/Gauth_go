package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/examples/cascade/pkg/mesh"
)

func main() {
	fmt.Println("\nStarting Cascading Failures Demonstration")
	fmt.Println("----------------------------------------")
	fmt.Println("Initial Configuration:")
	fmt.Println("- 7 interconnected services")
	fmt.Println("- Complex dependency graph")
	fmt.Println("- Circuit breakers, rate limits, and bulkheads")
	fmt.Println("----------------------------------------")

	serviceMesh := mesh.NewServiceMesh()
	ctx := context.Background()

	// Channel for controlling load monitoring
	loadUpdates := make(chan struct{})

	// Start background health monitoring
	go monitorHealth(serviceMesh, loadUpdates)

	// Run simulation phases
	runSimulationPhases(ctx, serviceMesh)

	// Clean up
	close(loadUpdates)
	time.Sleep(time.Second) // Allow final metrics to print

	// Print final report
	serviceMesh.PrintHealthReport()
}

func monitorHealth(serviceMesh *mesh.ServiceMesh, done chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			serviceMesh.PrintHealthReport()
		case <-done:
			return
		}
	}
}

func runSimulationPhases(ctx context.Context, serviceMesh *mesh.ServiceMesh) {
	phases := []struct {
		loadFactor float64
		services   []mesh.ServiceType
	}{
		{0.25, []mesh.ServiceType{mesh.PaymentService, mesh.OrderService, mesh.InventoryService}},
		{0.50, []mesh.ServiceType{mesh.PaymentService, mesh.OrderService, mesh.InventoryService}},
		{0.75, []mesh.ServiceType{mesh.PaymentService, mesh.OrderService, mesh.InventoryService}},
		{1.00, []mesh.ServiceType{mesh.PaymentService, mesh.OrderService, mesh.InventoryService}},
	}

	for phaseNum, phase := range phases {
		fmt.Printf("\nPhase %d: Load Factor %.2f\n", phaseNum+1, phase.loadFactor)

		// Set load factors for critical services
		for _, svcType := range phase.services {
			if err := serviceMesh.SetServiceLoad(svcType, phase.loadFactor); err != nil {
				log.Printf("Warning: failed to set service load for %v: %v", svcType, err)
			}
		}

		simulateTraffic(ctx, serviceMesh, 50)
		time.Sleep(2 * time.Second) // Pause between phases
	}
}

func simulateTraffic(ctx context.Context, serviceMesh *mesh.ServiceMesh, numClients int) {
	var wg sync.WaitGroup

	for client := 1; client <= numClients; client++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			// Simulate a transaction starting with OrderService
			if svc, exists := serviceMesh.GetService(mesh.OrderService); exists {
				start := time.Now()
				err := svc.ProcessRequest(ctx, serviceMesh)
				duration := time.Since(start)

				if err != nil {
					fmt.Printf("[Client %d] Failed after %v: %v\n",
						clientID, duration.Round(time.Millisecond), err)
				} else {
					fmt.Printf("[Client %d] Completed in %v\n",
						clientID, duration.Round(time.Millisecond))
				}
			}

			time.Sleep(100 * time.Millisecond)
		}(client)
	}

	wg.Wait()
}
