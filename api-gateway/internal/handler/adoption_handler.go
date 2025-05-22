package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhandarbeks/petstore-final-project/api-gateway/internal/client" // Adjust import path
	pbAdoption "github.com/zhandarbeks/petstore-final-project/genprotos/adoption" // Adjust import path
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	// "google.golang.org/protobuf/types/known/timestamppb" // If converting strings to timestamp for gRPC
)

// AdoptionHandler handles HTTP requests related to adoption applications.
type AdoptionHandler struct {
	adoptionClient client.AdoptionServiceClient // gRPC client for the adoption service
}

// NewAdoptionHandler creates a new AdoptionHandler.
func NewAdoptionHandler(adoptionClient client.AdoptionServiceClient) *AdoptionHandler {
	return &AdoptionHandler{adoptionClient: adoptionClient}
}

// CreateAdoptionApplication godoc
// @Summary Create a new adoption application
// @Description Submits an application to adopt a pet. Requires authentication.
// @Tags adoptions
// @Accept json
// @Produce json
// @Param application body pbAdoption.CreateAdoptionApplicationRequest true "Adoption application details"
// @Security BearerAuth
// @Success 201 {object} pbAdoption.AdoptionApplicationResponse "Successfully created adoption application"
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /adoptions [post]
func (h *AdoptionHandler) CreateAdoptionApplication(c *gin.Context) {
	var req pbAdoption.CreateAdoptionApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.UserId == "" || req.PetId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID and Pet ID are required"})
		return
	}

	grpcCtx := c.Request.Context()
	resp, err := h.adoptionClient.CreateAdoptionApplication(grpcCtx, &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			case codes.FailedPrecondition: 
				c.JSON(http.StatusConflict, gin.H{"error": st.Message()})
			case codes.AlreadyExists: 
				c.JSON(http.StatusConflict, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create application: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create application: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// GetAdoptionApplication godoc
// @Summary Get an adoption application by ID
// @Description Retrieves details of a specific adoption application. Requires authentication (applicant or admin).
// @Tags adoptions
// @Produce json
// @Param applicationId path string true "Application ID"
// @Security BearerAuth
// @Success 200 {object} pbAdoption.AdoptionApplicationResponse "Successfully retrieved application"
// @Failure 400 {object} map[string]string "Invalid application ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Application not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /adoptions/{applicationId} [get]
func (h *AdoptionHandler) GetAdoptionApplication(c *gin.Context) {
	appID := c.Param("applicationId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application ID is required"})
		return
	}

	grpcCtx := c.Request.Context()
	req := &pbAdoption.GetAdoptionApplicationRequest{ApplicationId: appID}
	resp, err := h.adoptionClient.GetAdoptionApplication(grpcCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get application: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get application: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateAdoptionApplicationStatus godoc
// @Summary Update an adoption application's status
// @Description Updates the status of an adoption application (e.g., by an admin). Requires authentication.
// @Tags adoptions
// @Accept json
// @Produce json
// @Param applicationId path string true "Application ID"
// @Param statusUpdate body pbAdoption.UpdateAdoptionApplicationStatusRequest true "Status update details"
// @Security BearerAuth
// @Success 200 {object} pbAdoption.AdoptionApplicationResponse "Successfully updated application status"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (e.g., not admin)"
// @Failure 404 {object} map[string]string "Application not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /adoptions/{applicationId}/status [patch]
func (h *AdoptionHandler) UpdateAdoptionApplicationStatus(c *gin.Context) {
	appID := c.Param("applicationId")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application ID is required in path"})
		return
	}

	var reqBody pbAdoption.UpdateAdoptionApplicationStatusRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	reqBody.ApplicationId = appID 

	grpcCtx := c.Request.Context()
	resp, err := h.adoptionClient.UpdateAdoptionApplicationStatus(grpcCtx, &reqBody)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update application status: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update application status: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListUserAdoptionApplications godoc
// @Summary List adoption applications for a user
// @Description Retrieves all adoption applications submitted by the authenticated user. Requires authentication.
// @Tags adoptions
// @Produce json
// @Param userId path string true "User ID (must match authenticated user)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param status_filter query string false "Filter by application status (PENDING_REVIEW, APPROVED, REJECTED, CANCELLED_BY_USER)"
// @Security BearerAuth
// @Success 200 {object} pbAdoption.ListAdoptionApplicationsResponse "Successfully retrieved applications"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/{userId}/adoptions [get]
func (h *AdoptionHandler) ListUserAdoptionApplications(c *gin.Context) {
	userID := c.Param("userId") 
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required in path"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	statusFilterStr := c.Query("status_filter")

	pageVal, err := strconv.ParseInt(pageStr, 10, 32)
	if err != nil || pageVal < 1 {
		pageVal = 1
	}
	limitVal, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limitVal < 1 {
		limitVal = 10
	}

	pageInt32 := int32(pageVal)
	limitInt32 := int32(limitVal)

	req := &pbAdoption.ListUserAdoptionApplicationsRequest{
		UserId: userID,
		Page:   &pageInt32,  // Pass pointer
		Limit:  &limitInt32, // Pass pointer
	}

	if statusFilterStr != "" {
		if val, ok := pbAdoption.ApplicationStatus_value[statusFilterStr]; ok {
			statusEnum := pbAdoption.ApplicationStatus(val)
			req.StatusFilter = &statusEnum // Pass pointer
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status_filter value"})
			return
		}
	}

	grpcCtx := c.Request.Context()
	resp, err := h.adoptionClient.ListUserAdoptionApplications(grpcCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list applications: " + st.Message()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list applications: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}