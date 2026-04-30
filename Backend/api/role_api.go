package api

import (
	"errors"
	"net/http"

	"my-course-backend/service"

	"github.com/gin-gonic/gin"
)

// AssignRoleInput represents the Data Transfer Object (DTO) for role reassignment.
// 
// Validation Schema:
// - UserID: Targeted user's internal ID. Mandatory to avoid ambiguous updates.
// - RoleName: Targeted role identifier. Mandatory to ensure a valid target state.
// 
// Reflection & Tags:
// Uses `json` tags for unmarshaling the request body and `binding` tags for Gin's 
// internal Validator (v10) engine to enforce non-zero/non-empty values.
type AssignRoleInput struct {
	UserID   uint   `json:"user_id" binding:"required"`
	RoleName string `json:"role_name" binding:"required"`
}

// requireSuperManagerRole serves as a specialized Security Interceptor.
//
// Role-Based Access Control (RBAC) Workflow:
// This function implements the "Principle of Least Privilege" (PoLP). 
// By centralizing the SuperManager check here, we ensure that administrative 
// power is not leaked to lower-level accounts (Managers or Instructors).
//
// Authentication Flow:
// 1. Header Parsing: Locates the JWT in the 'Authorization' header.
// 2. Token Introspection: Service-layer calls verify the signature and expiry.
// 3. Claims Verification: Specifically looks for the 'RoleID' claim.
//
// Design Pattern - Early Return:
// It uses the "Early Return" pattern to stop execution as soon as an 
// authorization violation is detected, preventing nested 'if-else' ladders.
func requireSuperManagerRole(c *gin.Context) error {
	// getTokenStringFromAuthHeader is expected to strip the "Bearer " prefix 
	// and return the raw Base64-encoded JWT string.
	tokenString, err := getTokenStringFromAuthHeader(c)
	if err != nil {
		// HTTP 401 Unauthorized: The request lacks valid authentication credentials.
		// Use gin.H for a quick, map-based JSON response.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication token missing or invalid"})
		return err
	}

	// Service Layer Delegation:
	// We call the service layer to decode and validate the JWT.
	// Hardcoding '2' as the SuperManager ID is a common shortcut for internal tools, 
	// though in enterprise systems, this would be checked against a constant or DB.
	roleID, err := service.GetRoleIDFromToken(tokenString)
	if err != nil || roleID != 2 {
		// HTTP 403 Forbidden: The server understood the request but refuses to authorize it.
		// This distinguishes between "Who are you?" (401) and "You aren't allowed here" (403).
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: This action requires SuperManager privileges"})
		return errors.New("insufficient permissions")
	}

	// Handshake verified; control is returned to the main handler.
	return nil
}

// AssignUserRole is the primary Controller endpoint for modifying user permissions.
//
// Endpoint Path: [PATCH/POST] /admin/assign-role
//
// Request Lifecycle & Error Handling Strategy:
//
// Step 1: Authorization Guarding
// We first verify the identity of the requester. This prevents unauthorized users 
// from even triggering the JSON parsing logic (Defense in Depth).
//
// Step 2: JSON Schema Validation
// We bind the incoming request stream to the AssignRoleInput struct. 
// If the frontend sends a string where an integer is expected, or misses a 
// required field, Gin returns a 400 Bad Request immediately.
//
// Step 3: Service Layer Invocation
// The controller remains "thin" by delegating business logic to service.AssignUserRole.
// This allows the logic to be reused in CLI tools or other API versions.
//
// Step 4: Success Notification
// On a successful update, we return a 200 OK. Standardizing these messages 
// helps frontend developers build consistent toast/notification components.
func AssignUserRole(c *gin.Context) {
	// Security Check: Block non-SuperManagers.
	if err := requireSuperManagerRole(c); err != nil {
		// The error response has already been written to the ResponseWriter 
		// inside requireSuperManagerRole, so we simply terminate the execution.
		return
	}

	var input AssignRoleInput
	// Context.ShouldBindJSON: A convenience method to unmarshal JSON into a struct.
	// It automatically sets the request body reader back to its start if needed 
	// and handles multiple content types.
	if err := c.ShouldBindJSON(&input); err != nil {
		// Log the error internally and return a descriptive message to the client.
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Business Logic execution:
	// We pass the validated UserID and the desired RoleName.
	if err := service.AssignUserRole(input.UserID, input.RoleName); err != nil {
		// If the service fails (e.g., target user doesn't exist, or role name 
		// typo like "Admiin"), we return a 400 Bad Request.
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to update user role",
			"details": err.Error(),
		})
		return
	}

	// Success response:
	// Inform the client that the database state has been successfully modified.
	c.JSON(http.StatusOK, gin.H{
		"message": "User role updated successfully",
		"target_user_id": input.UserID,
		"new_role": input.RoleName,
	})
}