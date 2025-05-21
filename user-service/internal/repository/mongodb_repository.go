package repository

import (
	"context"
	"errors"
	"log"
	// "time" // No longer needed here

	"github.com/zhandarbeks/petstore-final-project/user-service/internal/domain" // Adjust import path
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive" // Still needed for ID generation
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongoUserRepository is the MongoDB implementation of UserRepository
type mongoUserRepository struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

// NewMongoDBUserRepository creates a new instance of mongoUserRepository.
func NewMongoDBUserRepository(ctx context.Context, uri, dbName, collectionName string) (UserRepository, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Error connecting to MongoDB: %v", err)
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("Error pinging MongoDB: %v", err)
		if dErr := client.Disconnect(context.Background()); dErr != nil {
			log.Printf("Error disconnecting MongoDB after ping failure: %v", dErr)
		}
		return nil, err
	}
	log.Println("Successfully connected to MongoDB!")

	db := client.Database(dbName)
	collection := db.Collection(collectionName)

	indexModels := []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
		// Note: If _id is stored as a string, MongoDB automatically indexes it.
		// If you were storing it as ObjectID and wanted to ensure the string version was also indexed for other queries,
		// you might add an index on a separate string ID field, but that's not our case here if _id itself is the string.
	}
	_, err = collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		log.Printf("Warning: Could not create indexes for collection %s: %v", collectionName, err)
	} else {
		log.Printf("Indexes ensured for collection %s", collectionName)
	}

	return &mongoUserRepository{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

func (r *mongoUserRepository) Close(ctx context.Context) error {
	if r.client != nil {
		log.Println("Disconnecting MongoDB client...")
		return r.client.Disconnect(ctx)
	}
	return nil
}

func (r *mongoUserRepository) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	// Ensure ID is a hex string. If not provided, generate one.
	// This hex string will be stored as the _id field.
	if user.ID == "" {
		user.ID = primitive.NewObjectID().Hex()
	} else {
		// Validate if the provided ID is a valid hex, if not, it might cause issues or be stored as is.
		// For simplicity, we assume if an ID is provided, it's intended to be the _id.
		// If it needs to be an ObjectID hex, validation could be added.
		// For now, we let it be, as MongoDB will store it as a string if it's not a convertible ObjectID
		// when the struct field is string.
	}
	user.PrepareForCreate()

	_, err := r.collection.InsertOne(ctx, user) // user.ID (string) will be used for _id
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errors.New("user with this email or username already exists")
		}
		log.Printf("Error creating user in MongoDB: %v", err)
		return nil, err
	}
	return user, nil
}

// GetUserByID retrieves a user by their ID (which is a string).
func (r *mongoUserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	// Since domain.User.ID is string and likely stored as string for _id,
	// we query directly with the string id.
	var user domain.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user) // Query with string id
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		log.Printf("Error getting user by ID '%s' from MongoDB: %v", id, err)
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by their email address.
func (r *mongoUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found with this email")
		}
		log.Printf("Error getting user by email from MongoDB: %v", err)
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates an existing user's details using their string ID.
func (r *mongoUserRepository) UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	// user.ID is the string ID to match against the _id field in MongoDB.
	if user.ID == "" {
		return nil, errors.New("user ID cannot be empty for update")
	}
	user.PrepareForUpdate()

	update := bson.M{
		"$set": bson.M{
			"username":   user.Username,
			"full_name":  user.FullName,
			"updated_at": user.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": user.ID}, update) // Query with string user.ID
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errors.New("cannot update user, username or email may conflict")
		}
		log.Printf("Error updating user '%s' in MongoDB: %v", user.ID, err)
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, errors.New("user not found for update")
	}
	// If ModifiedCount is 0 but MatchedCount is 1, it means the data was the same.
	// Return the user object that was passed in (it has the updated timestamp).
	return user, nil
}

// DeleteUser removes a user from the database using their string ID.
func (r *mongoUserRepository) DeleteUser(ctx context.Context, id string) error {
	// id is the string ID to match against the _id field in MongoDB.
	if id == "" {
		return errors.New("user ID cannot be empty for delete")
	}
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id}) // Query with string id
	if err != nil {
		log.Printf("Error deleting user '%s' from MongoDB: %v", id, err)
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("user not found for deletion")
	}
	return nil
}