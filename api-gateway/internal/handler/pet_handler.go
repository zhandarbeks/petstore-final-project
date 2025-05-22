package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhandarbeks/petstore-final-project/api-gateway/internal/client" // Adjust import path
	pbPet "github.com/zhandarbeks/petstore-final-project/genprotos/pet"       // Adjust import path
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
)

// PetHandler handles HTTP requests related to pets.
type PetHandler struct {
	petClient client.PetServiceClient // gRPC client for the pet service
}

// NewPetHandler creates a new PetHandler.
func NewPetHandler(petClient client.PetServiceClient) *PetHandler {
	return &PetHandler{petClient: petClient}
}

// CreatePet godoc
// @Summary Create a new pet listing
// @Description Adds a new pet to the store. Requires authentication.
// @Tags pets
// @Accept json
// @Produce json
// @Param pet body pbPet.CreatePetRequest true "Pet details"
// @Security BearerAuth
// @Success 201 {object} pbPet.PetResponse "Successfully created pet"
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /pets [post]
func (h *PetHandler) CreatePet(c *gin.Context) {
	var req pbPet.CreatePetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Example: Get authenticated user ID from context (set by auth middleware)
	// userID, exists := c.Get("userID")
	// if !exists {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	// 	return
	// }
	// req.ListedByUserId = userID.(string) // Set the user who is listing the pet

	grpcCtx := c.Request.Context()
	resp, err := h.petClient.CreatePet(grpcCtx, &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pet: " + st.Message()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pet: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// GetPet godoc
// @Summary Get a pet by ID
// @Description Retrieves details of a specific pet.
// @Tags pets
// @Produce json
// @Param petId path string true "Pet ID"
// @Success 200 {object} pbPet.PetResponse "Successfully retrieved pet"
// @Failure 400 {object} map[string]string "Invalid pet ID"
// @Failure 404 {object} map[string]string "Pet not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /pets/{petId} [get]
func (h *PetHandler) GetPet(c *gin.Context) {
	petID := c.Param("petId")
	if petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pet ID is required"})
		return
	}

	grpcCtx := c.Request.Context()
	req := &pbPet.GetPetRequest{PetId: petID}
	resp, err := h.petClient.GetPet(grpcCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pet: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pet: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UpdatePet godoc
// @Summary Update a pet's details
// @Description Updates information for an existing pet. Requires authentication.
// @Tags pets
// @Accept json
// @Produce json
// @Param petId path string true "Pet ID"
// @Param pet body pbPet.UpdatePetRequest true "Pet update details"
// @Security BearerAuth
// @Success 200 {object} pbPet.PetResponse "Successfully updated pet"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (e.g., not owner)"
// @Failure 404 {object} map[string]string "Pet not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /pets/{petId} [patch]
func (h *PetHandler) UpdatePet(c *gin.Context) {
	petID := c.Param("petId")
	if petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pet ID is required in path"})
		return
	}

	var req pbPet.UpdatePetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	req.PetId = petID // Ensure PetId from path is used

	grpcCtx := c.Request.Context()
	resp, err := h.petClient.UpdatePet(grpcCtx, &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pet: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pet: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

// DeletePet godoc
// @Summary Delete a pet listing
// @Description Deletes a pet. Requires authentication.
// @Tags pets
// @Produce json
// @Param petId path string true "Pet ID"
// @Security BearerAuth
// @Success 204 "Successfully deleted pet"
// @Failure 400 {object} map[string]string "Invalid pet ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden (e.g., not owner)"
// @Failure 404 {object} map[string]string "Pet not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /pets/{petId} [delete]
func (h *PetHandler) DeletePet(c *gin.Context) {
	petID := c.Param("petId")
	if petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pet ID is required"})
		return
	}

	grpcCtx := c.Request.Context()
	req := &pbPet.DeletePetRequest{PetId: petID}
	_, err := h.petClient.DeletePet(grpcCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pet: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pet: " + err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

// ListPets godoc
// @Summary List available pets
// @Description Retrieves a list of pets, with optional filters and pagination.
// @Tags pets
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param species_filter query string false "Filter by species"
// @Param status_filter query string false "Filter by adoption status (AVAILABLE, PENDING_ADOPTION, ADOPTED)"
// @Success 200 {object} pbPet.ListPetsResponse "Successfully retrieved list of pets"
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /pets [get]
func (h *PetHandler) ListPets(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	speciesFilterQuery := c.Query("species_filter")
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

	req := &pbPet.ListPetsRequest{
		Page:  &pageInt32,  // Pass pointer
		Limit: &limitInt32, // Pass pointer
	}

	if speciesFilterQuery != "" {
		req.SpeciesFilter = &speciesFilterQuery // Pass pointer
	}

	if statusFilterStr != "" {
		if val, ok := pbPet.AdoptionStatus_value[statusFilterStr]; ok {
			statusEnum := pbPet.AdoptionStatus(val)
			req.StatusFilter = &statusEnum // Pass pointer
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status_filter value. Valid values: AVAILABLE, PENDING_ADOPTION, ADOPTED"})
			return
		}
	}

	grpcCtx := c.Request.Context()
	resp, err := h.petClient.ListPets(grpcCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list pets: " + st.Message()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list pets: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UpdatePetAdoptionStatus godoc
// @Summary Update a pet's adoption status
// @Description Updates the adoption status of a pet. Requires authentication (e.g. admin or involved user).
// @Tags pets
// @Accept json
// @Produce json
// @Param petId path string true "Pet ID"
// @Param statusUpdate body pbPet.UpdatePetAdoptionStatusRequest true "Adoption status update details"
// @Security BearerAuth
// @Success 200 {object} pbPet.PetResponse "Successfully updated pet adoption status"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Pet not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /pets/{petId}/status [patch]
func (h *PetHandler) UpdatePetAdoptionStatus(c *gin.Context) {
	petID := c.Param("petId")
	if petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pet ID is required in path"})
		return
	}

	var reqBody pbPet.UpdatePetAdoptionStatusRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	reqBody.PetId = petID // Ensure PetId from path is used in the gRPC request

	grpcCtx := c.Request.Context()
	resp, err := h.petClient.UpdatePetAdoptionStatus(grpcCtx, &reqBody)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pet status: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pet status: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}