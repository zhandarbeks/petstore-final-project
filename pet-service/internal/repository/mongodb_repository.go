package repository

import (
	"context"
	"errors"
	"log"
	"time" // Added import for time

	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/domain" // Adjust import path
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoPetRepository struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// NewMongoDBPetRepository creates a new instance of mongoPetRepository.
func NewMongoDBPetRepository(ctx context.Context, uri, dbName, collectionName string) (PetRepository, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Pet Service | Error connecting to MongoDB: %v", err)
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("Pet Service | Error pinging MongoDB: %v", err)
		if dErr := client.Disconnect(context.Background()); dErr != nil {
			log.Printf("Pet Service | Error disconnecting MongoDB after ping failure: %v", dErr)
		}
		return nil, err
	}
	log.Println("Pet Service | Successfully connected to MongoDB!")

	db := client.Database(dbName)
	collection := db.Collection(collectionName)

	// Example indexes for pets collection
	indexModels := []mongo.IndexModel{
		{Keys: bson.D{{Key: "species", Value: 1}}},
		{Keys: bson.D{{Key: "adoption_status", Value: 1}}},
		{Keys: bson.D{{Key: "age", Value: 1}}},
		// Add more indexes based on common query patterns
	}
	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		log.Printf("Pet Service | Warning: Could not create indexes for collection %s: %v", collectionName, err)
	} else {
		log.Printf("Pet Service | Indexes ensured for collection %s", collectionName)
	}

	return &mongoPetRepository{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

func (r *mongoPetRepository) Close(ctx context.Context) error {
	if r.client != nil {
		log.Println("Pet Service | Disconnecting MongoDB client...")
		return r.client.Disconnect(ctx)
	}
	return nil
}

func (r *mongoPetRepository) CreatePet(ctx context.Context, pet *domain.Pet) (*domain.Pet, error) {
	if pet.ID == "" {
		pet.ID = primitive.NewObjectID().Hex()
	}
	pet.PrepareForCreate() // Sets CreatedAt, UpdatedAt, default AdoptionStatus

	_, err := r.collection.InsertOne(ctx, pet)
	if err != nil {
		log.Printf("Pet Service | Error creating pet in MongoDB: %v", err)
		return nil, err
	}
	return pet, nil
}

func (r *mongoPetRepository) GetPetByID(ctx context.Context, id string) (*domain.Pet, error) {
	var pet domain.Pet
	// Assuming _id in MongoDB is stored as the string hex itself, as per domain.Pet.ID
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&pet)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("pet not found")
		}
		log.Printf("Pet Service | Error getting pet by ID '%s' from MongoDB: %v", id, err)
		return nil, err
	}
	return &pet, nil
}

func (r *mongoPetRepository) UpdatePet(ctx context.Context, pet *domain.Pet) (*domain.Pet, error) {
	if pet.ID == "" {
		return nil, errors.New("pet ID cannot be empty for update")
	}
	pet.PrepareForUpdate() // Sets UpdatedAt

	// Construct update document. Only update fields that are meant to be updatable.
	updateFields := bson.M{
		"name":             pet.Name,
		"species":          pet.Species,
		"breed":            pet.Breed,
		"age":              pet.Age,
		"description":      pet.Description,
		"adoption_status":  pet.AdoptionStatus,
		"image_urls":       pet.ImageURLs, // Overwrites the entire array
		"listed_by_user_id": pet.ListedByUserID,
		"adopted_by_user_id": pet.AdoptedByUserID,
		"updated_at":       pet.UpdatedAt,
	}
	// If you want partial updates for ImageURLs (e.g., add/remove), that would require different logic.

	update := bson.M{"$set": updateFields}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": pet.ID}, update)
	if err != nil {
		log.Printf("Pet Service | Error updating pet '%s' in MongoDB: %v", pet.ID, err)
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, errors.New("pet not found for update")
	}
	return pet, nil // Return the updated pet object
}

func (r *mongoPetRepository) DeletePet(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("pet ID cannot be empty for delete")
	}
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		log.Printf("Pet Service | Error deleting pet '%s' from MongoDB: %v", id, err)
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("pet not found for deletion")
	}
	return nil
}

func (r *mongoPetRepository) ListPets(ctx context.Context, page, limit int, filters map[string]interface{}) ([]*domain.Pet, int64, error) {
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
	// findOptions.SetSort(bson.D{{"created_at", -1}}) // Example sort by creation date descending

	// Build query from filters
	query := bson.M{}
	if filters != nil {
		for key, value := range filters {
			// Basic equality filter. For more complex filters (ranges, regex), add more logic.
			// Ensure filter keys match BSON field names.
			query[key] = value
		}
	}

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		log.Printf("Pet Service | Error listing pets from MongoDB: %v", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var pets []*domain.Pet
	if err = cursor.All(ctx, &pets); err != nil {
		log.Printf("Pet Service | Error decoding listed pets from MongoDB: %v", err)
		return nil, 0, err
	}

	totalCount, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		log.Printf("Pet Service | Error counting pets in MongoDB: %v", err)
		return nil, 0, err
	}

	return pets, totalCount, nil
}

func (r *mongoPetRepository) UpdatePetAdoptionStatus(ctx context.Context, id string, newStatus domain.AdoptionStatus, adopterUserID *string) (*domain.Pet, error) {
	if id == "" {
		return nil, errors.New("pet ID cannot be empty for status update")
	}
	if !domain.IsValidAdoptionStatus(newStatus) {
		return nil, errors.New("invalid new adoption status provided")
	}

	updateFields := bson.M{
		"adoption_status": newStatus,
		"updated_at":      time.Now().UTC(), // time.Now() is used here
	}
	if adopterUserID != nil && *adopterUserID != "" && newStatus == domain.StatusAdopted {
		updateFields["adopted_by_user_id"] = *adopterUserID
	} else if newStatus != domain.StatusAdopted {
		// If status is changing away from adopted, clear the adopted_by_user_id
		updateFields["adopted_by_user_id"] = "" // or use $unset if you prefer to remove the field
	}


	update := bson.M{"$set": updateFields}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		log.Printf("Pet Service | Error updating pet adoption status for ID '%s': %v", id, err)
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, errors.New("pet not found for status update")
	}

	// Fetch and return the updated pet
	return r.GetPetByID(ctx, id)
}