package handler

import (
	"context"
	"errors" // This will be used now, or removed if not. Let's check usage.
	"log"
	// "time" // Removed, as direct time operations might not be needed here if timestamppb handles all.

	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/domain"   // Adjust import path
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/usecase" // Adjust import path
	pb "github.com/zhandarbeks/petstore-final-project/genprotos/adoption"            // Adjust import path to your generated protos

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb" // For converting time.Time to/from protobuf Timestamp
)

// AdoptionHandler implements the gRPC service for adoption application operations.
type AdoptionHandler struct {
	pb.UnimplementedAdoptionServiceServer // Embed for forward compatibility
	usecase                             usecase.AdoptionUsecase
}

// NewAdoptionHandler creates a new AdoptionHandler.
func NewAdoptionHandler(uc usecase.AdoptionUsecase) *AdoptionHandler {
	if uc == nil {
		log.Fatal("AdoptionUsecase cannot be nil in NewAdoptionHandler")
	}
	return &AdoptionHandler{usecase: uc}
}

// --- Helper Functions for Type Conversion ---

func pbApplicationStatusToDomain(pbStatus pb.ApplicationStatus) domain.ApplicationStatus {
	switch pbStatus {
	case pb.ApplicationStatus_PENDING_REVIEW:
		return domain.StatusAppPendingReview
	case pb.ApplicationStatus_APPROVED:
		return domain.StatusAppApproved
	case pb.ApplicationStatus_REJECTED:
		return domain.StatusAppRejected
	case pb.ApplicationStatus_CANCELLED_BY_USER:
		return domain.StatusAppCancelledByUser
	default: // Handles APPLICATION_STATUS_UNSPECIFIED and any other unknown values
		return domain.StatusAppUnspecified
	}
}

func domainApplicationStatusToPb(domainStatus domain.ApplicationStatus) pb.ApplicationStatus {
	switch domainStatus {
	case domain.StatusAppPendingReview:
		return pb.ApplicationStatus_PENDING_REVIEW
	case domain.StatusAppApproved:
		return pb.ApplicationStatus_APPROVED
	case domain.StatusAppRejected:
		return pb.ApplicationStatus_REJECTED
	case domain.StatusAppCancelledByUser:
		return pb.ApplicationStatus_CANCELLED_BY_USER
	default:
		return pb.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED
	}
}

func domainAdoptionApplicationToPb(da *domain.AdoptionApplication) *pb.AdoptionApplication {
	if da == nil {
		return nil
	}
	var createdAtProto *timestamppb.Timestamp
	if !da.CreatedAt.IsZero() { // time.Time's IsZero() method is used
		createdAtProto = timestamppb.New(da.CreatedAt)
	}

	var updatedAtProto *timestamppb.Timestamp
	if !da.UpdatedAt.IsZero() { // time.Time's IsZero() method is used
		updatedAtProto = timestamppb.New(da.UpdatedAt)
	}

	return &pb.AdoptionApplication{
		Id:                da.ID,
		UserId:            da.UserID,
		PetId:             da.PetID,
		Status:            domainApplicationStatusToPb(da.Status),
		ApplicationNotes:  da.ApplicationNotes,
		ReviewNotes:       da.ReviewNotes,
		CreatedAt:         createdAtProto,
		UpdatedAt:         updatedAtProto,
	}
}

// --- gRPC Method Implementations ---

func (h *AdoptionHandler) CreateAdoptionApplication(ctx context.Context, req *pb.CreateAdoptionApplicationRequest) (*pb.AdoptionApplicationResponse, error) {
	log.Printf("Adoption Service | gRPC CreateAdoptionApplication request received for UserID: %s, PetID: %s", req.GetUserId(), req.GetPetId())

	if req.GetUserId() == "" || req.GetPetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "User ID and Pet ID are required")
	}

	reqData := usecase.CreateAdoptionApplicationRequestData{
		UserID:           req.GetUserId(),
		PetID:            req.GetPetId(),
		ApplicationNotes: req.GetApplicationNotes(),
	}

	createdApp, err := h.usecase.CreateAdoptionApplication(ctx, reqData)
	if err != nil {
		log.Printf("Adoption Service | Error during CreateAdoptionApplication usecase call: %v", err)
		// Example: Map specific domain errors if needed
		// Using errors.Is for better error checking if usecase returns wrapped errors or defined error types.
		if err.Error() == "pet is not available for adoption" { // Assuming usecase might return this specific string
			return nil, status.Errorf(codes.FailedPrecondition, err.Error())
		}
		if err.Error() == "active adoption application for this pet by this user already exists" { // Assuming usecase might return this
			return nil, status.Errorf(codes.AlreadyExists, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Failed to create adoption application: %v", err)
	}

	log.Printf("Adoption Service | Adoption application created successfully via gRPC: ID %s", createdApp.ID)
	return &pb.AdoptionApplicationResponse{Application: domainAdoptionApplicationToPb(createdApp)}, nil
}

func (h *AdoptionHandler) GetAdoptionApplication(ctx context.Context, req *pb.GetAdoptionApplicationRequest) (*pb.AdoptionApplicationResponse, error) {
	log.Printf("Adoption Service | gRPC GetAdoptionApplication request received for ID: %s", req.GetApplicationId())

	if req.GetApplicationId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Application ID is required")
	}

	app, err := h.usecase.GetAdoptionApplicationByID(ctx, req.GetApplicationId())
	if err != nil {
		log.Printf("Adoption Service | Error during GetAdoptionApplicationByID usecase call for ID %s: %v", req.GetApplicationId(), err)
		if err.Error() == "adoption application not found" { // Match error string from usecase/repo
			return nil, status.Errorf(codes.NotFound, "Adoption application not found")
		}
		return nil, status.Errorf(codes.Internal, "Failed to get adoption application: %v", err)
	}

	log.Printf("Adoption Service | Adoption application retrieved successfully via gRPC: ID %s", app.ID)
	return &pb.AdoptionApplicationResponse{Application: domainAdoptionApplicationToPb(app)}, nil
}

func (h *AdoptionHandler) UpdateAdoptionApplicationStatus(ctx context.Context, req *pb.UpdateAdoptionApplicationStatusRequest) (*pb.AdoptionApplicationResponse, error) {
	log.Printf("Adoption Service | gRPC UpdateAdoptionApplicationStatus request for ID: %s, NewStatus: %s", req.GetApplicationId(), req.GetNewStatus().String())

	if req.GetApplicationId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Application ID is required")
	}

	domainStatus := pbApplicationStatusToDomain(req.GetNewStatus())
	if domainStatus == domain.StatusAppUnspecified && req.GetNewStatus() != pb.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid new application status provided in request")
	}

	reqData := usecase.UpdateAdoptionApplicationStatusRequestData{
		NewStatus:   domainStatus,
		ReviewNotes: req.GetReviewNotes(),
	}

	updatedApp, err := h.usecase.UpdateAdoptionApplicationStatus(ctx, req.GetApplicationId(), reqData)
	if err != nil {
		log.Printf("Adoption Service | Error during UpdateAdoptionApplicationStatus usecase call for ID %s: %v", req.GetApplicationId(), err)
		// Using errors.Is for potentially wrapped errors from usecase/repo
		if errors.Is(err, errors.New("adoption application not found for status update")) || errors.Is(err, errors.New("adoption application not found")) {
			return nil, status.Errorf(codes.NotFound, "Adoption application not found for status update")
		}
		if errors.Is(err, errors.New("invalid new application status")) { // Assuming usecase returns this
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, errors.New("adopter user ID is required when setting status to ADOPTED")) { // Assuming usecase returns this
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Failed to update application status: %v", err)
	}

	log.Printf("Adoption Service | Adoption application status updated successfully via gRPC: ID %s to %s", updatedApp.ID, updatedApp.Status)
	return &pb.AdoptionApplicationResponse{Application: domainAdoptionApplicationToPb(updatedApp)}, nil
}

func (h *AdoptionHandler) ListUserAdoptionApplications(ctx context.Context, req *pb.ListUserAdoptionApplicationsRequest) (*pb.ListAdoptionApplicationsResponse, error) {
	log.Printf("Adoption Service | gRPC ListUserAdoptionApplications request for UserID: %s, Page: %d, Limit: %d, StatusFilter: %s",
		req.GetUserId(), req.GetPage(), req.GetLimit(), req.GetStatusFilter().String())

	if req.GetUserId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "User ID is required")
	}

	page := int(req.GetPage())
	limit := int(req.GetLimit())
	if page == 0 { page = 1 }
	if limit == 0 { limit = 10 }

	var statusFilter *domain.ApplicationStatus
	if req.GetStatusFilter() != pb.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED {
		ds := pbApplicationStatusToDomain(req.GetStatusFilter())
		// Check if the conversion resulted in Unspecified due to an invalid proto enum value
		if ds == domain.StatusAppUnspecified && req.GetStatusFilter() != pb.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid status filter value provided")
		}
		statusFilter = &ds
	}

	domainApps, totalCount, err := h.usecase.ListUserAdoptionApplications(ctx, req.GetUserId(), page, limit, statusFilter)
	if err != nil {
		log.Printf("Adoption Service | Error during ListUserAdoptionApplications usecase call: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to list user adoption applications: %v", err)
	}

	pbApps := make([]*pb.AdoptionApplication, len(domainApps))
	for i, da := range domainApps {
		pbApps[i] = domainAdoptionApplicationToPb(da)
	}

	log.Printf("Adoption Service | Listed %d adoption applications for UserID %s, total available: %d", len(pbApps), req.GetUserId(), totalCount)
	return &pb.ListAdoptionApplicationsResponse{
		Applications: pbApps,
		TotalCount:   int32(totalCount),
		Page:         int32(page),
		Limit:        int32(limit),
	}, nil
}