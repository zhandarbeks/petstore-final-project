package handler

import (
	"context"
	"errors"
	"log"
	"time" // Import time for RFC3339 formatting

	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/domain"   // Adjust import path
	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/usecase" // Adjust import path
	pb "github.com/zhandarbeks/petstore-final-project/genprotos/pet"            // Adjust import path to your generated protos

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	// "google.golang.org/protobuf/types/known/timestamppb" // No longer needed here if formatting to string directly
)

// PetHandler implements the gRPC service for pet operations.
type PetHandler struct {
	pb.UnimplementedPetServiceServer // Embed for forward compatibility
	usecase                        usecase.PetUsecase
}

// NewPetHandler creates a new PetHandler.
func NewPetHandler(uc usecase.PetUsecase) *PetHandler {
	if uc == nil {
		log.Fatal("PetUsecase cannot be nil in NewPetHandler")
	}
	return &PetHandler{usecase: uc}
}

// --- Helper Functions for Type Conversion ---

func pbAdoptionStatusToDomain(pbStatus pb.AdoptionStatus) domain.AdoptionStatus {
	switch pbStatus {
	case pb.AdoptionStatus_AVAILABLE:
		return domain.StatusAvailable
	case pb.AdoptionStatus_PENDING_ADOPTION:
		return domain.StatusPendingAdoption
	case pb.AdoptionStatus_ADOPTED:
		return domain.StatusAdopted
	default:
		return domain.StatusUnspecified
	}
}

func domainAdoptionStatusToPb(domainStatus domain.AdoptionStatus) pb.AdoptionStatus {
	switch domainStatus {
	case domain.StatusAvailable:
		return pb.AdoptionStatus_AVAILABLE
	case domain.StatusPendingAdoption:
		return pb.AdoptionStatus_PENDING_ADOPTION
	case domain.StatusAdopted:
		return pb.AdoptionStatus_ADOPTED
	default:
		return pb.AdoptionStatus_ADOPTION_STATUS_UNSPECIFIED
	}
}

func domainPetToPbPet(dp *domain.Pet) *pb.Pet {
	if dp == nil {
		return nil
	}
	var createdAtStr, updatedAtStr string
	if !dp.CreatedAt.IsZero() {
		createdAtStr = dp.CreatedAt.Format(time.RFC3339) // Corrected: Format to RFC3339 string
	}
	if !dp.UpdatedAt.IsZero() {
		updatedAtStr = dp.UpdatedAt.Format(time.RFC3339) // Corrected: Format to RFC3339 string
	}

	return &pb.Pet{
		Id:                dp.ID,
		Name:              dp.Name,
		Species:           dp.Species,
		Breed:             dp.Breed,
		Age:               dp.Age,
		Description:       dp.Description,
		AdoptionStatus:    domainAdoptionStatusToPb(dp.AdoptionStatus),
		ListedByUserId:    dp.ListedByUserID,
		AdoptedByUserId:   dp.AdoptedByUserID,
		ImageUrls:         dp.ImageURLs,
		CreatedAt:         createdAtStr, // Now a standard ISO string
		UpdatedAt:         updatedAtStr, // Now a standard ISO string
	}
}

// --- gRPC Method Implementations ---
// (Rest of the handler methods remain the same as previously defined)
// ... (CreatePet, GetPet, UpdatePet, DeletePet, ListPets, UpdatePetAdoptionStatus methods) ...

func (h *PetHandler) CreatePet(ctx context.Context, req *pb.CreatePetRequest) (*pb.PetResponse, error) {
	log.Printf("Pet Service | gRPC CreatePet request received for name: %s", req.GetName())

	if req.GetName() == "" || req.GetSpecies() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Pet name and species are required")
	}

	reqData := usecase.CreatePetRequestData{
		Name:           req.GetName(),
		Species:        req.GetSpecies(),
		Breed:          req.GetBreed(),
		Age:            req.GetAge(),
		Description:    req.GetDescription(),
		ListedByUserID: req.GetListedByUserId(),
		ImageURLs:      req.GetImageUrls(),
	}

	createdPet, err := h.usecase.CreatePet(ctx, reqData)
	if err != nil {
		log.Printf("Pet Service | Error during CreatePet usecase call for name %s: %v", req.GetName(), err)
		return nil, status.Errorf(codes.Internal, "Failed to create pet: %v", err)
	}

	log.Printf("Pet Service | Pet created successfully via gRPC: %s (ID: %s)", createdPet.Name, createdPet.ID)
	return &pb.PetResponse{Pet: domainPetToPbPet(createdPet)}, nil
}

func (h *PetHandler) GetPet(ctx context.Context, req *pb.GetPetRequest) (*pb.PetResponse, error) {
	log.Printf("Pet Service | gRPC GetPet request received for ID: %s", req.GetPetId())

	if req.GetPetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Pet ID is required")
	}

	pet, err := h.usecase.GetPetByID(ctx, req.GetPetId())
	if err != nil {
		log.Printf("Pet Service | Error during GetPetByID usecase call for ID %s: %v", req.GetPetId(), err)
		if err.Error() == "pet not found" {
			return nil, status.Errorf(codes.NotFound, "Pet not found")
		}
		return nil, status.Errorf(codes.Internal, "Failed to get pet: %v", err)
	}

	log.Printf("Pet Service | Pet retrieved successfully via gRPC: %s (ID: %s)", pet.Name, pet.ID)
	return &pb.PetResponse{Pet: domainPetToPbPet(pet)}, nil
}

func (h *PetHandler) UpdatePet(ctx context.Context, req *pb.UpdatePetRequest) (*pb.PetResponse, error) {
	log.Printf("Pet Service | gRPC UpdatePet request received for ID: %s", req.GetPetId())

	if req.GetPetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Pet ID is required for update")
	}

	reqData := usecase.UpdatePetRequestData{
		ImageURLs: req.GetImageUrls(), 
	}

	if req.GetName() != "" { 
		name := req.GetName()
		reqData.Name = &name
	}
	if req.GetSpecies() != "" {
		species := req.GetSpecies()
		reqData.Species = &species
	}
	
	breed := req.GetBreed() 
	reqData.Breed = &breed  

	if req.GetAge() != 0 { 
		age := req.GetAge()
		reqData.Age = &age
	}
	if req.GetDescription() != "" {
		desc := req.GetDescription()
		reqData.Description = &desc
	}
	
	if reqData.Name == nil && reqData.Species == nil && reqData.Breed == nil && reqData.Age == nil && reqData.Description == nil && req.ImageUrls == nil {
		 log.Println("Pet Service | UpdatePet: No fields provided for update")
		 return nil, status.Errorf(codes.InvalidArgument, "At least one field must be provided for update")
	}


	updatedPet, err := h.usecase.UpdatePet(ctx, req.GetPetId(), reqData)
	if err != nil {
		log.Printf("Pet Service | Error during UpdatePet usecase call for ID %s: %v", req.GetPetId(), err)
		if err.Error() == "pet not found for update" || err.Error() == "pet not found" {
			return nil, status.Errorf(codes.NotFound, "Pet not found for update")
		}
		return nil, status.Errorf(codes.Internal, "Failed to update pet: %v", err)
	}

	log.Printf("Pet Service | Pet updated successfully via gRPC: %s (ID: %s)", updatedPet.Name, updatedPet.ID)
	return &pb.PetResponse{Pet: domainPetToPbPet(updatedPet)}, nil
}

func (h *PetHandler) DeletePet(ctx context.Context, req *pb.DeletePetRequest) (*pb.EmptyResponse, error) {
	log.Printf("Pet Service | gRPC DeletePet request received for ID: %s", req.GetPetId())

	if req.GetPetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Pet ID is required for deletion")
	}

	err := h.usecase.DeletePet(ctx, req.GetPetId())
	if err != nil {
		log.Printf("Pet Service | Error during DeletePet usecase call for ID %s: %v", req.GetPetId(), err)
		if err.Error() == "pet not found for deletion" || err.Error() == "pet not found" {
			return nil, status.Errorf(codes.NotFound, "Pet not found for deletion")
		}
		return nil, status.Errorf(codes.Internal, "Failed to delete pet: %v", err)
	}

	log.Printf("Pet Service | Pet deleted successfully via gRPC: ID %s", req.GetPetId())
	return &pb.EmptyResponse{}, nil
}

func (h *PetHandler) ListPets(ctx context.Context, req *pb.ListPetsRequest) (*pb.ListPetsResponse, error) {
	log.Printf("Pet Service | gRPC ListPets request received. Page: %d, Limit: %d, SpeciesFilter: %s, StatusFilter: %s",
		req.GetPage(), req.GetLimit(), req.GetSpeciesFilter(), req.GetStatusFilter().String())

	page := int(req.GetPage())
	limit := int(req.GetLimit())
	if page == 0 { page = 1 } 
	if limit == 0 { limit = 10 } 

	filters := make(map[string]interface{})
	if req.GetSpeciesFilter() != "" {
		filters["species"] = req.GetSpeciesFilter()
	}
	if req.GetStatusFilter() != pb.AdoptionStatus_ADOPTION_STATUS_UNSPECIFIED {
		filters["adoption_status"] = pbAdoptionStatusToDomain(req.GetStatusFilter())
	}

	domainPets, totalCount, err := h.usecase.ListPets(ctx, page, limit, filters)
	if err != nil {
		log.Printf("Pet Service | Error during ListPets usecase call: %v", err)
		if err.Error() == "invalid adoption_status filter value" {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Failed to list pets: %v", err)
	}

	pbPets := make([]*pb.Pet, len(domainPets))
	for i, dp := range domainPets {
		pbPets[i] = domainPetToPbPet(dp)
	}

	log.Printf("Pet Service | Listed %d pets, total available: %d", len(pbPets), totalCount)
	return &pb.ListPetsResponse{
		Pets:       pbPets,
		TotalCount: int32(totalCount), 
		Page:       int32(page),
		Limit:      int32(limit),
	}, nil
}

func (h *PetHandler) UpdatePetAdoptionStatus(ctx context.Context, req *pb.UpdatePetAdoptionStatusRequest) (*pb.PetResponse, error) {
	log.Printf("Pet Service | gRPC UpdatePetAdoptionStatus request received for ID: %s, NewStatus: %s", req.GetPetId(), req.GetNewStatus().String())

	if req.GetPetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Pet ID is required")
	}

	domainStatus := pbAdoptionStatusToDomain(req.GetNewStatus())
	if domainStatus == domain.StatusUnspecified && req.GetNewStatus() != pb.AdoptionStatus_ADOPTION_STATUS_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid new adoption status provided in request")
	}

	var adopterIDPtr *string
	if req.GetAdopterUserId() != "" {
		adopterID := req.GetAdopterUserId()
		adopterIDPtr = &adopterID
	}

	updatedPet, err := h.usecase.UpdatePetAdoptionStatus(ctx, req.GetPetId(), domainStatus, adopterIDPtr)
	if err != nil {
		log.Printf("Pet Service | Error during UpdatePetAdoptionStatus usecase call for ID %s: %v", req.GetPetId(), err)
		if errors.Is(err, errors.New("pet not found for status update")) || errors.Is(err, errors.New("pet not found")) {
			return nil, status.Errorf(codes.NotFound, "Pet not found for status update")
		}
		if errors.Is(err, errors.New("invalid new adoption status provided")) || errors.Is(err, errors.New("invalid new adoption status")) {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, errors.New("adopter user ID is required when setting status to ADOPTED")) {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Failed to update pet adoption status: %v", err)
	}

	log.Printf("Pet Service | Pet adoption status updated successfully via gRPC for ID: %s", updatedPet.ID)
	return &pb.PetResponse{Pet: domainPetToPbPet(updatedPet)}, nil
}