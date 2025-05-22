package handler

import (
	"net/http"
	"strings" // For parsing Bearer token

	"github.com/gin-gonic/gin"
	"github.com/zhandarbeks/petstore-final-project/api-gateway/internal/client" // Adjust import path
	pbUser "github.com/zhandarbeks/petstore-final-project/genprotos/user"       // Adjust import path
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
)

// UserHandler handles HTTP requests related to users.
type UserHandler struct {
	userClient client.UserServiceClient // gRPC client for the user service
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userClient client.UserServiceClient) *UserHandler {
	return &UserHandler{userClient: userClient}
}

// RegisterUser godoc
// @Summary Register a new user
// @Description Creates a new user account.
// @Tags users
// @Accept json
// @Produce json
// @Param user body pbUser.RegisterUserRequest true "User registration details"
// @Success 201 {object} pbUser.UserResponse "Successfully registered user"
// @Failure 400 {object} map[string]string "Invalid request payload or already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/register [post]
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req pbUser.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Basic validation (more can be added)
	if req.Username == "" || req.Email == "" || req.Password == "" || req.FullName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username, email, password, and full name are required"})
		return
	}

	grpcCtx := c.Request.Context() // Use context from Gin request for gRPC call
	resp, err := h.userClient.RegisterUser(grpcCtx, &req)
	if err != nil {
		// Try to map gRPC error codes to HTTP status codes
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			case codes.AlreadyExists:
				c.JSON(http.StatusConflict, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// LoginUser godoc
// @Summary Log in a user
// @Description Authenticates a user and returns an access token.
// @Tags users
// @Accept json
// @Produce json
// @Param credentials body pbUser.LoginUserRequest true "User login credentials"
// @Success 200 {object} pbUser.LoginUserResponse "Successfully logged in"
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/login [post]
func (h *UserHandler) LoginUser(c *gin.Context) {
	var req pbUser.LoginUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	grpcCtx := c.Request.Context()
	resp, err := h.userClient.LoginUser(grpcCtx, &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			case codes.Unauthenticated:
				c.JSON(http.StatusUnauthorized, gin.H{"error": st.Message()}) // "Invalid email or password"
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetUser godoc
// @Summary Get user profile
// @Description Retrieves the profile of a user by their ID.
// @Tags users
// @Produce json
// @Param userId path string true "User ID"
// @Security BearerAuth
// @Success 200 {object} pbUser.UserResponse "Successfully retrieved user profile"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/{userId} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Example: Extracting user ID from JWT (if auth middleware sets it)
	// authenticatedUserID, exists := c.Get("userID") // Assuming middleware sets "userID" in Gin context
	// if !exists {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Missing user ID from token"})
	// 	return
	// }
	// if userID != authenticatedUserID.(string) { // Basic check: can only get own profile, or implement admin logic
	//  c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You can only view your own profile"})
	//  return
	// }


	grpcCtx := c.Request.Context()
	req := &pbUser.GetUserRequest{UserId: userID}
	resp, err := h.userClient.GetUser(grpcCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateUserProfile godoc
// @Summary Update user profile
// @Description Updates the profile of the authenticated user.
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID (must match authenticated user)"
// @Param user body pbUser.UpdateUserProfileRequest true "User profile update details (only username and full_name can be updated)"
// @Security BearerAuth
// @Success 200 {object} pbUser.UserResponse "Successfully updated user profile"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/{userId} [patch]
func (h *UserHandler) UpdateUserProfile(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required in path"})
		return
	}

	// Example: Authorization check (ensure user can only update their own profile)
	// authenticatedUserID, exists := c.Get("userID") // Assuming middleware sets "userID"
	// if !exists || userID != authenticatedUserID.(string) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You can only update your own profile"})
	// 	return
	// }

	var req pbUser.UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	req.UserId = userID // Ensure UserId from path is used

	grpcCtx := c.Request.Context()
	resp, err := h.userClient.UpdateUserProfile(grpcCtx, &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

// DeleteUser godoc
// @Summary Delete user account
// @Description Deletes the account of the authenticated user.
// @Tags users
// @Produce json
// @Param userId path string true "User ID (must match authenticated user)"
// @Security BearerAuth
// @Success 204 "Successfully deleted user account"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/{userId} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Example: Authorization check
	// authenticatedUserID, exists := c.Get("userID")
	// if !exists || userID != authenticatedUserID.(string) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You can only delete your own account"})
	// 	return
	// }

	grpcCtx := c.Request.Context()
	req := &pbUser.DeleteUserRequest{UserId: userID}
	_, err := h.userClient.DeleteUser(grpcCtx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user: " + st.Message()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user: " + err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

// Helper to extract token from Authorization header (example)
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, "Bearer ")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}