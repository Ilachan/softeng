/**
 * Package api
 * * Overview:
 * This package serves as the 'Controller' or 'Transport' layer of the 
 * my-course-backend application. It is responsible for handling incoming 
 * HTTP requests, validating request schemas, and coordinating between the 
 * Service layer and the HTTP response writer.
 *
 * Architecture Principles:
 * 1. Thin Controllers: Logic is kept to a minimum; business rules reside in the Service layer.
 * 2. Status Code Semantic Correctness: Uses the full spectrum of HTTP 2xx, 4xx, and 5xx codes.
 * 3. Security First: Every administrative endpoint is guarded by RBAC checks.
 *
 * Package Dependencies:
 * - net/http: Standard library for HTTP status constants.
 * - strconv: For safe type casting of URL path parameters.
 * - model/service: Internal modules for business logic and data structures.
 * - gin-gonic/gin: The underlying high-performance HTTP framework.
 */
package api

import (
	"errors"
	"net/http"
	"strconv"
	"my-course-backend/model"
	"my-course-backend/service"

	"github.com/gin-gonic/gin"
)

/* ========================================================================
   SECTION: SECURITY MIDDLEWARE & AUTHORIZATION HELPERS
   ======================================================================== */

/**
 * requireManagerRole: An authorization guard for Manager-level resources.
 *
 * Identification Logic:
 * In this system, Role IDs are mapped as follows:
 * - Role 1: Student (Standard User)
 * - Role 2: Manager (Middle Management / Course Coordinator)
 * - Role 3: SuperManager (Full System Administrator)
 *
 * Functional Workflow:
 * 1. JWT Extraction: Retrieves the token from the request header.
 * 2. Token Validation: Calls the service layer to decrypt and verify the JWT signature.
 * 3. Role Inspection: Checks if the RoleID claims are either 2 or 3.
 *
 * Security Implications:
 * Failure to pass this check results in a 401 (Unauthenticated) or 403 (Unauthorized).
 * This prevents lateral movement where a regular student might attempt to modify
 * course content via a forged or compromised session.
 */
func requireManagerRole(c *gin.Context) error {
	// Step 1: Locate the Bearer token in the 'Authorization' header.
	// Common errors: Header missing, prefix 'Bearer ' missing, or malformed string.
	tokenString, err := getTokenStringFromAuthHeader(c)
	if err != nil {
		// Log: No token provided. Return 401.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Access denied: No authentication token found"})
		return err
	}

	// Step 2: Introspect the token to extract the role identifier.
	// This involves decoding the JWT claims and verifying the expiration (exp) time.
	roleID, err := service.GetRoleIDFromToken(tokenString)
	if err != nil {
		// Log: Token is expired or signature verification failed.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired: Please log in again"})
		return err
	}

	// Step 3: Enforce the RBAC (Role-Based Access Control) policy.
	// We use an OR condition to allow both standard Managers and SuperManagers.
	if roleID != 2 && roleID != 3 {
		// Log: User authenticated but lacks administrative clearance.
		// Return 403 Forbidden to signify a permission mismatch.
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: This resource requires a Manager-level account"})
		return errors.New("forbidden")
	}

	// Logic flow continues to the controller if no error is returned.
	return nil
}

/* ========================================================================
   SECTION: COURSE MANAGEMENT (CRUD OPERATIONS)
   ======================================================================== */

/**
 * ManagerCreateClass (POST /classes)
 *
 * Responsibility:
 * Facilitates the creation of a new academic course/class entry in the system.
 *
 * Request Lifecycle:
 * 1. Auth: Verify Manager/SuperManager role.
 * 2. Validation: Ensure the JSON body complies with the CourseUpsertInput schema.
 * 3. Persistence: The service layer generates the DB record and associated metadata.
 * 4. Response: Returns the newly created class object including the DB-generated ID.
 *
 * Scalability Note:
 * This endpoint uses 'ShouldBindJSON' which reads the request body and binds it
 * to a struct using reflection. This ensures strict typing before data reaches the service.
 */
func ManagerCreateClass(c *gin.Context) {
	// RBAC Guard Clause.
	if err := requireManagerRole(c); err != nil {
		// Response already written in the helper. Abort execution.
		return
	}

	// Prepare the DTO (Data Transfer Object) for input binding.
	var input service.CourseUpsertInput

	// Validates fields like 'Title', 'StartTime', and 'Capacity' based on struct tags.
	if err := c.ShouldBindJSON(&input); err != nil {
		// 400 Bad Request: Usually triggered by malformed JSON or missing required fields.
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Input validation failed",
			"details": err.Error(),
		})
		return
	}

	// Service Layer Delegation:
	// We pass the validated DTO to the service layer to handle the 'Create' logic.
	class, err := service.ManagerCreateCourse(input)
	if err != nil {
		// 500 Internal Server Error: Database failure or system-level exception.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "An unexpected error occurred while creating the class",
			"internal_message": err.Error(),
		})
		return
	}

	// 201 Created: Semantic success code for resource instantiation.
	c.JSON(http.StatusCreated, gin.H{
		"message": "Class created successfully",
		"class":   class,
	})
}

/**
 * ManagerUpdateClass (PUT /classes/:id)
 *
 * Responsibility:
 * Performs a full or partial update of an existing class record identified by the 'id' parameter.
 *
 * Implementation Details:
 * - ID Extraction: Uses c.Param("id") to retrieve the dynamic path variable.
 * - Parsing: Since path variables are strings, we use ParseUint to convert it for DB lookup.
 * - Error Mapping: Distinguishes between '404 Not Found' (bad ID) and '400 Bad Request' (bad data).
 *
 * Concurrency Note:
 * This logic relies on GORM's Update behavior. It's recommended to ensure the
 * service layer handles 'versioning' or 'locking' if concurrent updates are frequent.
 */
func ManagerUpdateClass(c *gin.Context) {
	// Guard the endpoint against unauthorized access.
	if err := requireManagerRole(c); err != nil {
		return
	}

	// Capture the target record ID from the URL string.
	idStr := c.Param("id")

	// Convert the string-based ID to an unsigned integer (base 10, 32-bit).
	// This is a safety measure against SQL injection via path parameters.
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The provided Class ID is malformed or invalid"})
		return
	}

	// Bind the update payload.
	var input service.CourseUpsertInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Call service to locate and modify the record.
	class, err := service.ManagerUpdateCourse(uint(id64), input)
	if err != nil {
		// Specific handling for 'Resource Missing' scenarios.
		if err.Error() == "class not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "No class found with the specified ID"})
			return
		}

		// Fallback for general server errors.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record: " + err.Error()})
		return
	}

	// 200 OK: Return the updated state of the resource.
	c.JSON(http.StatusOK, gin.H{
		"message": "Class details synchronized successfully",
		"class":   class,
	})
}

/**
 * ManagerDeleteClass (DELETE /classes/:id)
 *
 * Responsibility:
 * Removes a class record from the primary database.
 *
 * Impact Warning:
 * Deleting a class may have cascading effects on the 'Enrollment' and 'DailyActivity' tables.
 * The service layer should handle 'Soft Deletes' if data retention is required.
 *
 * Workflow:
 * 1. RBAC check.
 * 2. Path parameter extraction.
 * 3. Service execution.
 * 4. Response notification.
 */
func ManagerDeleteClass(c *gin.Context) {
	// Authorization Check.
	if err := requireManagerRole(c); err != nil {
		return
	}

	// Retrieve the Resource ID.
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource identifier format"})
		return
	}

	// Execute deletion.
	if err := service.ManagerDeleteCourse(uint(id64)); err != nil {
		// Handle 404 (Missing Record).
		if err.Error() == "class not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "The class you are attempting to delete does not exist"})
			return
		}

		// Handle 500 (Internal Failure).
		c.JSON(http.StatusInternalServerError, gin.H{"error": "System failure during record removal"})
		return
	}

	// 200 OK: Confirm deletion to the client.
	c.JSON(http.StatusOK, gin.H{
		"message": "Class and associated metadata have been successfully purged",
		"id":      id64,
	})
}



// ManagerRegister serves as the handler for high-privileged account creation.
//
// Endpoint Context:
// - Route: POST /auth/manager/register
// - Security Model: Open endpoint but requires a valid 'Invite Code' to prevent public abuse.
//
// Logic Breakdown:
// 1. DTO Binding: Unmarshals JSON into a structured ManagerRegisterInput.
// 2. Business Logic: Hands off data to the service layer where invite codes are validated 
//    and passwords are salted/hashed.
// 3. Error Classification: Uses a switch statement to translate domain-specific errors 
//    (from the service layer) into precise HTTP Status Codes for the client.
func ManagerRegister(c *gin.Context) {
    var input model.ManagerRegisterInput
    
    // Attempt to bind the request body. If the JSON structure is malformed or missing 
    // fields required by the model tags, we return a 400 Bad Request.
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Request payload validation failed",
            "details": err.Error(),
        })
        return
    }

    // Execute the registration process. This involves checking the invite code, 
    // verifying email uniqueness, and creating the database entry.
    if err := service.RegisterManager(input); err != nil {
        // Detailed Error Mapping: We map internal service error strings to 
        // semantic HTTP statuses (400, 403, 409, or 500).
        switch err.Error() {
        case "email already exists":
            // 409 Conflict: Indicates a resource already exists.
            c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        case "invalid invite code":
            // 400 Bad Request: Syntax or basic validation failed for the code.
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        case "invite code is not active", "invite code already used", "invite code expired", "invite code not allowed for this email":
            // 403 Forbidden: The code is known but the server refuses to allow its use.
            c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
        default:
            // 500 Internal Server Error: Catch-all for database or unexpected system failures.
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Manager registration failed: " + err.Error()})
        }
        return
    }

    // 201 Created: Standard success response for a successful resource creation.
    c.JSON(http.StatusCreated, gin.H{"message": "Administrative account created successfully"})
}

// ManagerListUsers provides a paginated view of the user directory for management staff.
// 
// Access Control:
// - Role Required: Manager or higher (verified by requireManagerRole).
//
// Pagination Parameters (Query String):
// - page: The target page number (defaults to 1).
// - limit: Records per page (defaults to 20).
func ManagerListUsers(c *gin.Context) {
    // Security Guard: Ensure the current session holder has 'Manager' permissions.
    if err := requireManagerRole(c); err != nil {
        // The helper function handles the error response if authorization fails.
        return
    }

    // Parse 'page' and 'limit' from the query parameters.
    // Default values are provided to ensure the API doesn't fail if params are missing.
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

    // Invoke the service to fetch a subset of users and total counts for the frontend.
    // This allows the client to render pagination controls (e.g., "Page 1 of 5").
    users, total, page, limit, totalPages, err := service.ManagerListUsers(page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to fetch user list: " + err.Error()})
        return
    }

    // 200 OK: Return a comprehensive pagination metadata object alongside the data.
    c.JSON(http.StatusOK, gin.H{
        "users":       users,
        "page":        page,
        "limit":       limit,
        "total":       total,
        "total_pages": totalPages,
    })
}

// ManagerListUserEnrollments retrieves all course registrations for a specific student.
//
// URL Parameters:
// - id: The unique primary key of the User whose enrollments are being inspected.
func ManagerListUserEnrollments(c *gin.Context) {
    // Verify Managerial authorization levels.
    if err := requireManagerRole(c); err != nil {
        return
    }

    // Extract the dynamic ':id' segment from the URL path.
    idStr := c.Param("id")
    // Convert the string ID to an unsigned 32-bit integer.
    id64, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Numeric user ID is required in the path segment"})
        return
    }

    // Fetch enrollment records. The service layer should handle course preloading.
    enrollments, err := service.ManagerListUserEnrollments(uint(id64))
    if err != nil {
        // Specifically handle the 404 case if the user ID doesn't exist in our system.
        if err.Error() == "user not found" {
            c.JSON(http.StatusNotFound, gin.H{"error": "Target user could not be found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database lookup failed: " + err.Error()})
        return
    }

    // Return the list of enrollments (may be an empty array if the user has no courses).
    c.JSON(http.StatusOK, gin.H{"enrollments": enrollments})
}

// ManagerAddEnrollmentInput is the DTO for manually adding a user to a course.
type ManagerAddEnrollmentInput struct {
    CourseID uint `json:"course_id" binding:"required"`
}

// ManagerAddUserEnrollment allows an administrator to bypass the self-enrollment flow 
// and manually register a student for a specific course.
//
// This is critical for administrative overrides (e.g., student support, waitlist management).
func ManagerAddUserEnrollment(c *gin.Context) {
    // Enforce Manager role check.
    if err := requireManagerRole(c); err != nil {
        return
    }

    // Resolve User ID from the URL path.
    idStr := c.Param("id")
    id64, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Malformed User ID in URL"})
        return
    }

    // Resolve Course ID from the JSON request body.
    var input ManagerAddEnrollmentInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
        return
    }

    // Process the enrollment request.
    if err := service.ManagerAddUserEnrollment(uint(id64), input.CourseID); err != nil {
        // Multi-layered error handling to differentiate between missing resources (404) 
        // and logical conflicts (409).
        switch err.Error() {
        case "user not found", "class not found", "no upcoming session found for this class":
            c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        case "enrollment already exists", "class is full":
            c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        default:
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create enrollment: " + err.Error()})
        }
        return
    }

    // 201 Created: Successful manual enrollment.
    c.JSON(http.StatusCreated, gin.H{"message": "Manual enrollment successfully completed"})
}

// ManagerDeleteUserEnrollment handles the removal of a student's enrollment record.
//
// This endpoint utilizes a dual-parameter path:
// - /manager/users/:id/enrollments/:course_id
// where :id is the student and :course_id is the course to be dropped.
func ManagerDeleteUserEnrollment(c *gin.Context) {
    // Ensure the caller has appropriate management clearance.
    if err := requireManagerRole(c); err != nil {
        return
    }

    // Extract User ID from path segment.
    userIDStr := c.Param("id")
    userID64, err := strconv.ParseUint(userIDStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID format"})
        return
    }

    // Extract Course ID from path segment.
    courseIDStr := c.Param("course_id")
    courseID64, err := strconv.ParseUint(courseIDStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Course ID format"})
        return
    }

    // Perform the deletion via the service layer.
    if err := service.ManagerDeleteUserEnrollment(uint(userID64), uint(courseID64)); err != nil {
        // 404 Error: If the specific User-Course combination doesn't exist in the enrollment table.
        if err.Error() == "enrollment not found" {
            c.JSON(http.StatusNotFound, gin.H{"error": "Matching enrollment record not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete operation failed: " + err.Error()})
        return
    }

    // 200 OK: Confirmation of deletion.
    c.JSON(http.StatusOK, gin.H{"message": "User successfully unenrolled from the course"})
}
/* ========================================================================
   FOOTER: ARCHITECTURAL GUIDELINES
   ========================================================================
   1. All JSON responses must follow the { "error": "msg" } or { "data": obj } pattern.
   2. Ensure 'strconv' conversions match the database column types (uint32/uint64).
   3. The 'requireManagerRole' should eventually be moved to a Middleware for better DRY.
   ======================================================================== */