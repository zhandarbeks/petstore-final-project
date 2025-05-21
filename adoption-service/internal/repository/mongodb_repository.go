package repository

import (
	"context"
	"errors"
	"log"
	"time" // Required for UpdateAdoptionApplicationStatus

	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/domain" // Adjust import path
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoAdoptionRepository struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// NewMongoDBAdoptionRepository creates a new instance of mongoAdoptionRepository.
func NewMongoDBAdoptionRepository(ctx context.Context, uri, dbName, collectionName string) (AdoptionRepository, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Adoption Service | Error connecting to MongoDB: %v", err)
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("Adoption Service | Error pinging MongoDB: %v", err)
		if dErr := client.Disconnect(context.Background()); dErr != nil {
			log.Printf("Adoption Service | Error disconnecting MongoDB after ping failure: %v", dErr)
		}
		return nil, err
	}
	log.Println("Adoption Service | Successfully connected to MongoDB!")

	db := client.Database(dbName)
	collection := db.Collection(collectionName)

	// Example indexes for adoption_applications collection
	indexModels := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "pet_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		// Composite index for common query in ListAdoptionApplicationsByUserID
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}}},
	}
	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		log.Printf("Adoption Service | Warning: Could not create indexes for collection %s: %v", collectionName, err)
	} else {
		log.Printf("Adoption Service | Indexes ensured for collection %s", collectionName)
	}

	return &mongoAdoptionRepository{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

func (r *mongoAdoptionRepository) Close(ctx context.Context) error {
	if r.client != nil {
		log.Println("Adoption Service | Disconnecting MongoDB client...")
		return r.client.Disconnect(ctx)
	}
	return nil
}

func (r *mongoAdoptionRepository) CreateAdoptionApplication(ctx context.Context, app *domain.AdoptionApplication) (*domain.AdoptionApplication, error) {
	if app.ID == "" {
		app.ID = primitive.NewObjectID().Hex()
	}
	app.PrepareForCreate() // Sets CreatedAt, UpdatedAt, default Status

	// Optional: Check if an active application already exists for this user and pet
	// filter := bson.M{"user_id": app.UserID, "pet_id": app.PetID, "status": bson.M{"$in": []domain.ApplicationStatus{domain.StatusAppPendingReview, domain.StatusAppApproved}}}
	// count, err := r.collection.CountDocuments(ctx, filter)
	// if err != nil {
	// 	log.Printf("Adoption Service | Error checking existing application: %v", err)
	// 	return nil, errors.New("failed to verify existing applications")
	// }
	// if count > 0 {
	// 	return nil, errors.New("active adoption application for this pet by this user already exists")
	// }


	_, err := r.collection.InsertOne(ctx, app)
	if err != nil {
		log.Printf("Adoption Service | Error creating adoption application in MongoDB: %v", err)
		return nil, err
	}
	return app, nil
}

func (r *mongoAdoptionRepository) GetAdoptionApplicationByID(ctx context.Context, id string) (*domain.AdoptionApplication, error) {
	var app domain.AdoptionApplication
	// Assuming _id in MongoDB is stored as the string hex itself
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&app)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("adoption application not found")
		}
		log.Printf("Adoption Service | Error getting adoption application by ID '%s' from MongoDB: %v", id, err)
		return nil, err
	}
	return &app, nil
}

func (r *mongoAdoptionRepository) UpdateAdoptionApplicationStatus(ctx context.Context, id string, newStatus domain.ApplicationStatus, reviewNotes string) (*domain.AdoptionApplication, error) {
	if id == "" {
		return nil, errors.New("application ID cannot be empty for status update")
	}
	if !domain.IsValidApplicationStatus(newStatus) {
		return nil, errors.New("invalid new application status provided")
	}

	updateFields := bson.M{
		"status":       newStatus,
		"review_notes": reviewNotes, // Update review notes regardless
		"updated_at":   time.Now().UTC(),
	}
	update := bson.M{"$set": updateFields}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		log.Printf("Adoption Service | Error updating application status for ID '%s': %v", id, err)
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, errors.New("adoption application not found for status update")
	}

	// Fetch and return the updated application
	return r.GetAdoptionApplicationByID(ctx, id)
}

func (r *mongoAdoptionRepository) ListAdoptionApplicationsByUserID(ctx context.Context, userID string, page, limit int, statusFilter *domain.ApplicationStatus) ([]*domain.AdoptionApplication, int64, error) {
	if userID == "" {
		return nil, 0, errors.New("user ID is required to list adoption applications")
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10 // Default limit
	}
	skip := (page - 1) * limit

	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(limit))
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}}) // Sort by newest first

	query := bson.M{"user_id": userID}
	if statusFilter != nil && *statusFilter != "" && *statusFilter != domain.StatusAppUnspecified {
		if !domain.IsValidApplicationStatus(*statusFilter) {
			return nil, 0, errors.New("invalid status filter value")
		}
		query["status"] = *statusFilter
	}

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		log.Printf("Adoption Service | Error listing adoption applications by UserID '%s': %v", userID, err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var applications []*domain.AdoptionApplication
	if err = cursor.All(ctx, &applications); err != nil {
		log.Printf("Adoption Service | Error decoding listed adoption applications: %v", err)
		return nil, 0, err
	}

	totalCount, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		log.Printf("Adoption Service | Error counting adoption applications for UserID '%s': %v", userID, err)
		return nil, 0, err
	}

	return applications, totalCount, nil
}