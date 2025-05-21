package consumer

import (
	"context"
	"encoding/json"
	"log"
	"sync" // For managing goroutines during shutdown
	"time"

	"github.com/nats-io/nats.go"
	// You'll need to define these event structs based on what adoption-service publishes
	// For example:
	// "github.com/zhandarbeks/petstore-final-project/notification-service/internal/service"
)

// AdoptionApplicationCreatedEvent represents the data structure for this event.
// This should match the payload published by adoption-service.
type AdoptionApplicationCreatedEvent struct {
	EventType      string    `json:"event_type"`
	ApplicationID  string    `json:"application_id"`
	UserID         string    `json:"user_id"`
	PetID          string    `json:"pet_id"`
	Status         string    `json:"status"` // Consider using domain.ApplicationStatus type if shared
	AppliedAt      time.Time `json:"applied_at"`
}

// AdoptionApplicationStatusUpdatedEvent represents the data structure for this event.
type AdoptionApplicationStatusUpdatedEvent struct {
	EventType      string    `json:"event_type"`
	ApplicationID  string    `json:"application_id"`
	UserID         string    `json:"user_id"`
	PetID          string    `json:"pet_id"`
	NewStatus      string    `json:"new_status"` // Consider using domain.ApplicationStatus type
	UpdatedAt      time.Time `json:"updated_at"`
	ReviewNotes    string    `json:"review_notes"`
}

// EventHandler defines the interface for processing received NATS events.
// This will be implemented by your notification-service's core logic.
type EventHandler interface {
	HandleAdoptionApplicationCreated(ctx context.Context, event AdoptionApplicationCreatedEvent) error
	HandleAdoptionApplicationStatusUpdated(ctx context.Context, event AdoptionApplicationStatusUpdatedEvent) error
}

// NATSConsumer handles NATS subscriptions and message processing.
type NATSConsumer struct {
	nc           *nats.Conn
	js           nats.JetStreamContext // For JetStream, if used
	eventHandler EventHandler
	subscriptions []*nats.Subscription
	shutdownWg   sync.WaitGroup // WaitGroup for graceful shutdown of message handlers
	stopChan     chan struct{}    // Channel to signal goroutines to stop
}

// NewNATSConsumer creates a new NATS consumer.
func NewNATSConsumer(natsURL string, handler EventHandler) (*NATSConsumer, error) {
	if handler == nil {
		log.Fatal("Notification Service | FATAL: EventHandler cannot be nil for NATSConsumer")
	}

	nc, err := nats.Connect(natsURL, nats.Timeout(5*time.Second), nats.RetryOnFailedConnect(true), nats.MaxReconnects(10), nats.ReconnectWait(2*time.Second))
	if err != nil {
		log.Printf("Notification Service | Error connecting to NATS at %s: %v", natsURL, err)
		return nil, err
	}
	log.Printf("Notification Service | Successfully connected to NATS at %s", natsURL)

	// Optional: Initialize JetStream context if you plan to use durable subscriptions
	// js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	// if err != nil {
	// 	log.Printf("Notification Service | Error getting JetStream context: %v", err)
	// 	nc.Close()
	// 	return nil, err
	// }
	// log.Println("Notification Service | JetStream context obtained.")

	return &NATSConsumer{
		nc:           nc,
		// js:           js,
		eventHandler: handler,
		stopChan:     make(chan struct{}),
	}, nil
}

// StartSubscribers begins listening to configured NATS subjects.
func (c *NATSConsumer) StartSubscribers() error {
	log.Println("Notification Service | Starting NATS subscribers...")

	// Subscribe to AdoptionApplicationCreated events
	// For core NATS:
	subCreated, err := c.nc.Subscribe("adoption.application.created", c.handleCreatedMessage)
	// For JetStream (durable subscriber):
	// subCreated, err := c.js.Subscribe("adoption.application.created", c.handleCreatedMessage, nats.Durable("notification-service-created"), nats.AckNone())
	if err != nil {
		log.Printf("Notification Service | Error subscribing to 'adoption.application.created': %v", err)
		return err
	}
	c.subscriptions = append(c.subscriptions, subCreated)
	log.Println("Notification Service | Subscribed to 'adoption.application.created'")

	// Subscribe to AdoptionApplicationStatusUpdated events
	// For core NATS:
	subStatusUpdated, err := c.nc.Subscribe("adoption.application.status.updated", c.handleStatusUpdatedMessage)
	// For JetStream (durable subscriber):
	// subStatusUpdated, err := c.js.Subscribe("adoption.application.status.updated", c.handleStatusUpdatedMessage, nats.Durable("notification-service-status"), nats.AckNone())
	if err != nil {
		log.Printf("Notification Service | Error subscribing to 'adoption.application.status.updated': %v", err)
		return err
	}
	c.subscriptions = append(c.subscriptions, subStatusUpdated)
	log.Println("Notification Service | Subscribed to 'adoption.application.status.updated'")

	// Keep the main goroutine alive or manage via application lifecycle
	// For a simple worker, this might run indefinitely until Close() is called.
	// Or, StartSubscribers could return and main.go manages the lifecycle.
	// For this example, we'll assume main.go will call Close() on shutdown.

	return nil
}

func (c *NATSConsumer) handleCreatedMessage(msg *nats.Msg) {
	c.shutdownWg.Add(1)
	defer c.shutdownWg.Done()

	select {
	case <-c.stopChan:
		log.Printf("Notification Service | Shutting down handleCreatedMessage goroutine for subject: %s", msg.Subject)
		return
	default:
		log.Printf("Notification Service | Received message on subject '%s'", msg.Subject)
		var event AdoptionApplicationCreatedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Notification Service | Error unmarshalling AdoptionApplicationCreatedEvent: %v. Data: %s", err, string(msg.Data))
			// Optionally, send to a dead-letter queue or log more persistently
			return
		}

		// Process the event using the injected handler
		// Using a new context for each message processing, or pass one from a higher level.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Example timeout for processing
		defer cancel()

		if err := c.eventHandler.HandleAdoptionApplicationCreated(ctx, event); err != nil {
			log.Printf("Notification Service | Error handling AdoptionApplicationCreatedEvent for AppID %s: %v", event.ApplicationID, err)
			// Implement retry logic or dead-letter queue if necessary
		} else {
			log.Printf("Notification Service | Successfully processed AdoptionApplicationCreatedEvent for AppID %s", event.ApplicationID)
		}
	}
}

func (c *NATSConsumer) handleStatusUpdatedMessage(msg *nats.Msg) {
	c.shutdownWg.Add(1)
	defer c.shutdownWg.Done()

	select {
	case <-c.stopChan:
		log.Printf("Notification Service | Shutting down handleStatusUpdatedMessage goroutine for subject: %s", msg.Subject)
		return
	default:
		log.Printf("Notification Service | Received message on subject '%s'", msg.Subject)
		var event AdoptionApplicationStatusUpdatedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Notification Service | Error unmarshalling AdoptionApplicationStatusUpdatedEvent: %v. Data: %s", err, string(msg.Data))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := c.eventHandler.HandleAdoptionApplicationStatusUpdated(ctx, event); err != nil {
			log.Printf("Notification Service | Error handling AdoptionApplicationStatusUpdatedEvent for AppID %s: %v", event.ApplicationID, err)
		} else {
			log.Printf("Notification Service | Successfully processed AdoptionApplicationStatusUpdatedEvent for AppID %s", event.ApplicationID)
		}
	}
}

// Close gracefully shuts down the NATS consumer.
func (c *NATSConsumer) Close() {
	log.Println("Notification Service | Shutting down NATS consumer...")
	close(c.stopChan) // Signal message handling goroutines to stop

	for _, sub := range c.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Notification Service | Error unsubscribing from NATS subject '%s': %v", sub.Subject, err)
		} else {
			log.Printf("Notification Service | Unsubscribed from NATS subject '%s'", sub.Subject)
		}
	}

	// Drain ensures all messages in flight for client-side subscriptions are processed.
	// For JetStream, different semantics might apply for durable consumers.
	if c.nc != nil && !c.nc.IsClosed() {
		log.Println("Notification Service | Draining NATS connection...")
		if err := c.nc.Drain(); err != nil {
			log.Printf("Notification Service | Error draining NATS connection: %v", err)
		} else {
			log.Println("Notification Service | NATS connection drained.")
		}
		// nc.Close() // Drain will close the connection.
	}
	
	// Wait for message handlers to finish
	done := make(chan struct{})
	go func() {
		c.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Notification Service | All message handlers have completed.")
	case <-time.After(10 * time.Second): // Timeout for waiting
		log.Println("Notification Service | Timeout waiting for message handlers to complete.")
	}

	log.Println("Notification Service | NATS consumer shut down.")
}