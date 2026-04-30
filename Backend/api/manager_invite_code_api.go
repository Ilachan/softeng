package api

import (
	"net/http"

	"my-course-backend/model"
	"my-course-backend/service"

	"github.com/gin-gonic/gin"
)

// CreateManagerInviteCode serves as the entry point for generating new administrative credentials.
//
// HTTP Route Information:
// - Method: POST
// - Endpoint: /auth/manager/invite-codes
// - Authorization: Bearer Token required (SuperManager only)
//
// API Security Posture:
// This handler implements "Gatekeeper" logic. It performs multi-stage verification:
// 1. Token Existence: Ensures the request is authenticated via an Authorization header.
// 2. Identity Extraction: Resolves the unique UserID of the administrator performing the action.
// 3. RBAC Verification: Strictly enforces that only SuperManagers (RoleID 2) can generate codes.
//
// Data Flow Architecture:
// [Client Request] -> [Auth Middleware/Logic] -> [JSON Binding] -> [Service Layer] -> [Response]
func CreateManagerInviteCode(c *gin.Context) {
	/* -------------------- STEP 1: AUTHENTICATION HEADER RETRIEVAL -------------------- */

	// Extract the JSON Web Token (JWT) string from the HTTP 'Authorization' header.
	// Expected format: "Bearer <encoded_jwt_string>"
	tokenString, err := getTokenStringFromAuthHeader(c)
	if err != nil {
		// HTTP 401 Unauthorized: The request fails immediately if no identity token is found.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required: " + err.Error()})
		return
	}

	/* -------------------- STEP 2: TOKEN INTROSPECTION & IDENTITY -------------------- */

	// Parse the token claims to find the UserID of the requester.
	// This UserID is crucial for the "CreatedBy" audit trail in the database.
	userID, err := service.ExtractUserIDFromToken(tokenString)
	if err != nil {
		// HTTP 401 Unauthorized: This occurs if the token signature is invalid or the token has expired.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid identity token"})
		return
	}

	// Retrieve the RoleID associated with this token to perform permission checks.
	roleID, err := service.GetRoleIDFromToken(tokenString)
	if err != nil {
		// Safety fallback: if the role cannot be determined, treat the user as unauthenticated.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to resolve user privileges"})
		return
	}

	/* -------------------- STEP 3: ROLE-BASED ACCESS CONTROL (RBAC) -------------------- */

	// Only RoleID 2 (internally mapped to 'SuperManager') is permitted to escalate privileges
	// by creating new manager invitation codes.
	// This is a hard-coded security guard to prevent privilege escalation attacks.
	if roleID != 2 {
		// HTTP 403 Forbidden: The identity is valid (authenticated), but lacks the specific
		// clearance (authorized) required for this administrative resource.
		c.JSON(http.StatusForbidden, gin.H{"error": "Access Denied: Administrative (SuperManager) role required"})
		return
	}

	/* -------------------- STEP 4: REQUEST BODY PARSING & VALIDATION -------------------- */

	// Define a local instance of the expected JSON payload schema.
	var input model.CreateManagerInviteInput

	// ShouldBindJSON automatically maps the JSON body to the struct fields.
	// It performs structural validation (e.g., checking for required fields or correct data types).
	if err := c.ShouldBindJSON(&input); err != nil {
		// HTTP 400 Bad Request: Indicates the client provided malformed JSON or invalid parameters.
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data validation failure: " + err.Error()})
		return
	}

	/* -------------------- STEP 5: BUSINESS LOGIC EXECUTION (SERVICE) -------------------- */

	// Delegate the core logic to the Service Layer (Domain Layer).
	// This keeps the API layer "Thin" (only handling HTTP) and the Service layer "Fat" (handling logic).
	// service.CreateManagerInviteCode will handle random code generation and DB persistence.
	code, err := service.CreateManagerInviteCode(userID, input)
	if err != nil {
		// HTTP 500 Internal Server Error: This covers unexpected issues like database connectivity failures.
		// We return the error message for debugging, though in production, you might obfuscate this.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "System error during code generation: " + err.Error()})
		return
	}

	/* -------------------- STEP 6: RESPONSE DISPATCH -------------------- */

	// HTTP 201 Created: Signifies that a new resource (the invite code) was successfully instantiated.
	// The client receives the generated code string to be shared with the new manager.
	c.JSON(http.StatusCreated, gin.H{
		"message": "Security invite code generated successfully",
		"code":    code, // The alphanumeric or UUID string representing the invitation.
	})
}