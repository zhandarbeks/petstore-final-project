package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	// "sync" // Removed as it's not directly used in this main package
	"syscall"
	"time"

	// Adjust these import paths to match your project's module path and structure
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/client"
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/config"
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/consumer"
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/email"
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/service"
)

func main() {
	// 1. Load Notification Service Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Notification Service | FATAL: Error loading configuration: %v", err)
	}

	log.Println("Notification Service | Configuration loaded.")
	log.Printf("Notification Service | NATS URL: %s", cfg.NatsURL)
	log.Printf("Notification Service | SMTP Server: %s:%d", cfg.SMTPServer, cfg.SMTPPort)
	log.Printf("Notification Service | Sender Email: %s", cfg.SMTPSenderEmail)
	log.Printf("Notification Service | User Service gRPC URL: %s", cfg.UserServiceGRPCURL)
	log.Printf("Notification Service | Pet Service gRPC URL: %s", cfg.PetServiceGRPCURL)

	// Create a main context that can be used to signal shutdown
	mainCtx, cancelMainCtx := context.WithCancel(context.Background())
	defer cancelMainCtx()

	initTimeout := 15 * time.Second // Timeout for client initializations

	// 2. Initialize User Service gRPC Client
	userClientInitCtx, userClientCancel := context.WithTimeout(mainCtx, initTimeout)
	defer userClientCancel()
	userServiceClient, err := client.NewUserServiceGRPCClient(userClientInitCtx, cfg.UserServiceGRPCURL)
	if err != nil {
		log.Fatalf("Notification Service | FATAL: Failed to initialize User Service gRPC client: %v", err)
	}
	log.Println("Notification Service | User Service gRPC client initialized.")
	defer func() {
		log.Println("Notification Service | Closing User Service gRPC client connection...")
		if err := userServiceClient.Close(); err != nil {
			log.Printf("Notification Service | Error closing User Service gRPC client: %v", err)
		}
	}()

	// 3. Initialize Pet Service gRPC Client
	petClientInitCtx, petClientCancel := context.WithTimeout(mainCtx, initTimeout)
	defer petClientCancel()
	petServiceClient, err := client.NewPetServiceGRPCClient(petClientInitCtx, cfg.PetServiceGRPCURL)
	if err != nil {
		log.Fatalf("Notification Service | FATAL: Failed to initialize Pet Service gRPC client: %v", err)
	}
	log.Println("Notification Service | Pet Service gRPC client initialized.")
	defer func() {
		log.Println("Notification Service | Closing Pet Service gRPC client connection...")
		if err := petServiceClient.Close(); err != nil {
			log.Printf("Notification Service | Error closing Pet Service gRPC client: %v", err)
		}
	}()

	// 4. Initialize Email Sender
	emailSender, err := email.NewSMTPEmailSender(cfg.SMTPServer, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPSenderEmail)
	if err != nil {
		log.Fatalf("Notification Service | FATAL: Failed to initialize SMTP Email Sender: %v", err)
	}
	log.Println("Notification Service | SMTP Email Sender initialized.")

	// 5. Initialize Notification Service (which implements consumer.EventHandler)
	notificationSvc := service.NewNotificationService(emailSender, userServiceClient, petServiceClient)
	log.Println("Notification Service | Core notification service logic initialized.")

	// 6. Initialize NATS Consumer
	natsConsumer, err := consumer.NewNATSConsumer(cfg.NatsURL, notificationSvc)
	if err != nil {
		log.Fatalf("Notification Service | FATAL: Failed to initialize NATS consumer: %v", err)
	}
	log.Println("Notification Service | NATS consumer initialized.")

	// 7. Start NATS Subscribers
	if err := natsConsumer.StartSubscribers(); err != nil {
		log.Fatalf("Notification Service | FATAL: Failed to start NATS subscribers: %v", err)
	}
	log.Println("Notification Service | NATS subscribers started. Listening for events...")

	// 8. Wait for shutdown signal
	// This keeps the main goroutine alive while the NATS consumer works in the background.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Notification Service | Received signal: %v. Shutting down...", sig)

	// 9. Graceful Shutdown
	// Cancel the main context to signal other parts of the application if they use it.
	cancelMainCtx()

	// Close NATS consumer (which will unsubscribe and drain connections)
	natsConsumer.Close() // This method should handle waiting for message handlers to finish.

	// gRPC client connections are already deferred to close.

	log.Println("Notification Service | Shut down gracefully.")
}