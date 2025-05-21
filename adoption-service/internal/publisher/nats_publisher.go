package publisher

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/domain" // Adjust import path
)

// AdoptionEventPublisher defines the interface for publishing adoption-related events.
type AdoptionEventPublisher interface {
	PublishAdoptionApplicationCreated(ctx context.Context, app *domain.AdoptionApplication) error
	PublishAdoptionApplicationStatusUpdated(ctx context.Context, app *domain.AdoptionApplication) error
	Close()
}

// natsAdoptionPublisher is the NATS implementation of AdoptionEventPublisher.
type natsAdoptionPublisher struct {
	nc *nats.Conn // NATS connection
	// js nats.JetStreamContext // Uncomment if using NATS JetStream
}

// NewNATSAdoptionPublisher creates a new NATS publisher for adoption events.
func NewNATSAdoptionPublisher(natsURL string) (AdoptionEventPublisher, error) {
	nc, err := nats.Connect(natsURL, nats.Timeout(5*time.Second), nats.RetryOnFailedConnect(true), nats.MaxReconnects(3))
	if err != nil {
		log.Printf("Adoption Service | Error connecting to NATS at %s: %v", natsURL, err)
		return nil, err
	}
	log.Printf("Adoption Service | Successfully connected to NATS at %s", natsURL)

	// Uncomment and adjust if using NATS JetStream
	/*
		js, err := nc.JetStream()
		if err != nil {
			log.Printf("Adoption Service | Error getting JetStream context: %v", err)
			nc.Close()
			return nil, err
		}
		log.Println("Adoption Service | JetStream context obtained.")

		// Example: Ensure stream exists (idempotent)
		// _, err = js.AddStream(&nats.StreamConfig{
		// 	Name:     "ADOPTIONS",
		// 	Subjects: []string{"adoption.created", "adoption.status.updated"},
		// })
		// if err != nil {
		// 	log.Printf("Adoption Service | Error adding JetStream stream 'ADOPTIONS': %v", err)
		// 	// Decide if this is a fatal error or just a warning
		// }
	*/

	return &natsAdoptionPublisher{nc: nc /*, js: js */}, nil
}

// PublishAdoptionApplicationCreated publishes an event when a new adoption application is created.
func (p *natsAdoptionPublisher) PublishAdoptionApplicationCreated(ctx context.Context, app *domain.AdoptionApplication) error {
	subject := "adoption.application.created" // Define your NATS subject
	eventData := map[string]interface{}{
		"event_type":     "AdoptionApplicationCreated",
		"application_id": app.ID,
		"user_id":        app.UserID,
		"pet_id":         app.PetID,
		"status":         app.Status,
		"applied_at":     app.CreatedAt,
	}

	payload, err := json.Marshal(eventData)
	if err != nil {
		log.Printf("Adoption Service | Error marshalling AdoptionApplicationCreated event for app ID %s: %v", app.ID, err)
		return err
	}

	// Using core NATS publish
	err = p.nc.Publish(subject, payload)
	if err != nil {
		log.Printf("Adoption Service | Error publishing AdoptionApplicationCreated event to subject '%s' for app ID %s: %v", subject, app.ID, err)
		return err
	}

	// If using JetStream:
	// _, err = p.js.Publish(subject, payload, nats.Context(ctx))
	// if err != nil {
	// 	log.Printf("Adoption Service | Error publishing (JetStream) AdoptionApplicationCreated event to subject '%s' for app ID %s: %v", subject, app.ID, err)
	// 	return err
	// }

	log.Printf("Adoption Service | Published event to '%s' for new application ID: %s", subject, app.ID)
	return nil
}

// PublishAdoptionApplicationStatusUpdated publishes an event when an adoption application's status changes.
func (p *natsAdoptionPublisher) PublishAdoptionApplicationStatusUpdated(ctx context.Context, app *domain.AdoptionApplication) error {
	subject := "adoption.application.status.updated" // Define your NATS subject
	eventData := map[string]interface{}{
		"event_type":     "AdoptionApplicationStatusUpdated",
		"application_id": app.ID,
		"user_id":        app.UserID,
		"pet_id":         app.PetID,
		"new_status":     app.Status,
		"updated_at":     app.UpdatedAt,
		"review_notes":   app.ReviewNotes, // Include review notes if relevant
	}

	payload, err := json.Marshal(eventData)
	if err != nil {
		log.Printf("Adoption Service | Error marshalling AdoptionApplicationStatusUpdated event for app ID %s: %v", app.ID, err)
		return err
	}

	err = p.nc.Publish(subject, payload)
	if err != nil {
		log.Printf("Adoption Service | Error publishing AdoptionApplicationStatusUpdated event to subject '%s' for app ID %s: %v", subject, app.ID, err)
		return err
	}

	// If using JetStream:
	// _, err = p.js.Publish(subject, payload, nats.Context(ctx))
	// if err != nil {
	//  log.Printf("Adoption Service | Error publishing (JetStream) AdoptionApplicationStatusUpdated event to subject '%s' for app ID %s: %v", subject, app.ID, err)
	// 	return err
	// }

	log.Printf("Adoption Service | Published event to '%s' for application ID: %s, new status: %s", subject, app.ID, app.Status)
	return nil
}

// Close drains and closes the NATS connection.
func (p *natsAdoptionPublisher) Close() {
	if p.nc != nil {
		log.Println("Adoption Service | Draining and closing NATS connection...")
		p.nc.Drain() // Drains a connection for all subscribers and then closes it.
		p.nc.Close() // Redundant if Drain is used, but good for explicitness or if Drain fails.
		log.Println("Adoption Service | NATS connection closed.")
	}
}