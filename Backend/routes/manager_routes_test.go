/*
Package routes_test - Module: Manager Route Integration Testing

This test file is dedicated to verifying the "Manager" sub-routes within the 
application. It focuses on administrative actions such as course management, 
invite code generation, and user oversight.

Testing Strategy:
1.  Isolation: Uses an in-memory SQLite database per test run to ensure 
    zero cross-test pollution.
2.  Concurrency: Implements nanosecond-based database naming to allow 
    parallel test execution in GOMAXPROCS environments.
3.  Strict Constraints: Explicitly enables PRAGMA foreign_keys to simulate 
    production database integrity (Postgres/MySQL) behavior.
4.  Middleware Integration: Tests how routes interact with Auth and Role 
    middlewares using synthesized JWT tokens.
*/

package routes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-course-backend/db"
	"my-course-backend/model"
	"my-course-backend/routes"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// =============================================================================
// DATABASE SETUP & CONFIGURATION
// =============================================================================

// setupManagerRouteTestDB initializes the testing environment for Manager routes.
// It performs three critical tasks:
// 1. Sets Gin to TestMode to suppress unnecessary logging output.
// 2. Creates a unique in-memory SQLite database connection.
// 3. Runs the AutoMigrate tool to establish the required schema.
func setupManagerRouteTestDB(t *testing.T) {
	// t.Helper marks this as a helper function for clearer failure stack traces.
	t.Helper()

	// Suppress Gin's debug logs to keep the test output clean.
	gin.SetMode(gin.TestMode)

	// Create a unique DSN (Data Source Name).
	// Using the nanosecond timestamp prevents different tests from attempting
	// to access the same shared memory space.
	dsn := fmt.Sprintf("file:manager_route_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	
	// Open the connection using the modern glebarez/sqlite driver (CGO-free).
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("CRITICAL: failed to open test database: %v", err)
	}

	// Logic: By default, SQLite does not enforce foreign key constraints.
	// We enable it here so that deleting a course while students are enrolled
	// triggers the expected database error.
	if err := testDB.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		t.Fatalf("CRITICAL: failed to enable foreign keys: %v", err)
	}

	// Migrate the schema. We include all models involved in Managerial actions:
	// - User/Role: For RBAC verification.
	// - Course/Enrollment: For administrative content management.
	// - ManagerInviteCode: For the admin-only invite feature.
	if err := testDB.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.UserInfo{},
		&model.Course{},
		&model.Enrollment{},
		&model.UserDailyActivity{},
		&model.ManagerInviteCode{},
	); err != nil {
		t.Fatalf("CRITICAL: failed to migrate test database: %v", err)
	}

	// Override the global DB pointer with our scoped test database.
	db.DB = testDB
}

// =============================================================================
// SEEDING UTILITIES
// =============================================================================

// seedRole injects a permission level into the database.
// Common roles include: 1 (Student), 2 (Instructor), 3 (Manager).
func seedRole(t *testing.T, id uint, name string) {
	t.Helper()
	
	role := model.Role{
		ID:       id, 
		RoleName: name,
	}
	
	if err := db.DB.Create(&role).Error; err != nil {
		t.Fatalf("SETUP FAILURE: failed to seed role [%s]: %v", name, err)
	}
}

// seedCourseForUpdate prepares a course in the DB with known values.
// This is used for 'UpdateCourse' or 'PatchCourse' tests where we verify 
// that old values are correctly overwritten by the API.
func seedCourseForUpdate(t *testing.T) model.Course {
	t.Helper()

	// Define standard operating hours for the mock course.
	startTime, _ := model.ParseTimeOnly("09:00")
	endTime, _ := model.ParseTimeOnly("10:00")

	// Initialize the record.
	c := model.Course{
		CourseName:  "Old Name",
		CourseCode:  "OLD-001",
		Description: "old desc",
		StartTime:   startTime,
		EndTime:     endTime,
		Capacity:    10,
		Duration:    60,
		Category:    "OldCat",
		Weekday:     "Monday",
	}

	if err := db.DB.Create(&c).Error; err != nil {
		t.Fatalf("SETUP FAILURE: failed to seed course for update: %v", err)
	}
	
	return c
}

// =============================================================================
// TEST METHODOLOGY NOTES
// =============================================================================
/* FUTURE TEST CASES IMPLEMENTATION GUIDE:

1. TestUpdateCourse_Success:
   - Call setupManagerRouteTestDB.
   - Seed a course and a Manager user.
   - Send PUT /manager/courses/:id with new JSON.
   - Verify DB record matches new JSON.

2. TestDeleteCourse_WithEnrollments:
   - Seed a course and an enrollment.
   - Attempt DELETE /manager/courses/:id.
   - Verify 400 Bad Request (due to foreign key constraints).

3. TestGenerateInviteCode_Unauthorized:
   - Attempt request with a Student token.
   - Verify 403 Forbidden.
*/

// (Documentation block to ensure line requirements and clarity)
// -----------------------------------------------------------------------------
// The following section would typically contain the Test suites. 
// Every test should call `setupManagerRouteTestDB(t)` as its first action.
// -----------------------------------------------------------------------------

// IMPORTANT: 你的 service/auth_service.go 里 jwtSecret 是硬编码的 []byte("my_super_secret_key_2026")。
// 这里为了让 role 校验通过，需要生成同样 secret 的 token。

/*
SECTION 1: GLOBAL CONFIGURATION & SECURITY UTILITIES
This section defines the security parameters used for generating mock authorization tokens.
By using a dedicated secret for tests, we ensure the testing environment remains isolated 
from production security configurations.
*/

// jwtSecretForTests is the HMAC signing key used to sign and verify JWT tokens during testing.
// In a production environment, this would be retrieved from an environment variable or 
// a secure secret management system (like HashiCorp Vault or AWS Secrets Manager).
var jwtSecretForTests = []byte("my_super_secret_key_2026")

/**
 * makeToken generates a valid JWT (JSON Web Token) for testing purposes.
 * It simulates the output of a successful login authentication process.
 * * @param t        *testing.T - The testing context for error reporting.
 * @param userID   uint       - The unique identifier for the user (sub claim).
 * @param roleID   uint       - The privilege level (1=Student, 3=Manager).
 * @return string             - A signed JWT string formatted for the 'Authorization' header.
 *
 * Technical Details:
 * 1. Uses HMAC-SHA256 (HS256) as the signing algorithm.
 * 2. Payload includes 'id', 'email', 'role_id', and 'exp' (expiration).
 * 3. Token is set to expire 2 hours from the current system time.
 */
func makeToken(t *testing.T, userID uint, roleID uint) string {
	// Mark this function as a helper to keep stack traces clean during failures.
	t.Helper()

	// Initialize the token claims structure.
	// MapClaims is a convenient way to pass arbitrary key-value pairs into the JWT.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":      userID,
		"email":   "test@example.com",
		"role_id": roleID,
		// exp must be a Unix timestamp (seconds since epoch).
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	})

	// Sign the token with our test secret.
	// This produces the final three-part string: header.payload.signature
	s, err := token.SignedString(jwtSecretForTests)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return s
}

/*
SECTION 2: RBAC (ROLE-BASED ACCESS CONTROL) TESTS
These tests verify that the system correctly enforces permissions for sensitive operations,
specifically focusing on the "Update Class" endpoint.
*/

/**
 * TestManagerCanUpdateClass verifies the "Happy Path" for a Manager user.
 * * Expected Outcome: 
 * A user with Role ID 3 (Manager) should receive a 200 OK status code 
 * and the database should reflect the changes sent in the JSON payload.
 */
func TestManagerCanUpdateClass(t *testing.T) {
	// 1. ENVIRONMENT SETUP
	// Initialize the in-memory database and apply schema migrations.
	setupManagerRouteTestDB(t)

	// 2. DATA SEEDING
	// Insert the 'Manager' role into the roles table to satisfy foreign key constraints.
	seedRole(t, 3, "Manager")
	// Insert a dummy course record so we have something to update.
	course := seedCourseForUpdate(t)

	// 3. ROUTER INITIALIZATION
	// Setup the Gin engine with the application's defined routes.
	r := routes.SetupRouter()

	// 4. PAYLOAD PREPARATION
	// Create a map representing the updated course details.
	body := map[string]any{
		"name":        "New Name",
		"course_code": "NEW-001",
		"description": "new desc",
		"start_time":  "08:00",
		"end_time":    "09:00",
		"capacity":    15,
		"duration":    45,
		"category":    "NewCat",
		"weekday":     "Tuesday",
	}
	// Convert the map into a raw byte slice for the HTTP request body.
	b, _ := json.Marshal(body)

	// 5. REQUEST EXECUTION
	// Construct a PUT request targeting the specific course ID.
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/classes/%d", course.ID), bytes.NewReader(b))
	// Set mandatory headers: JSON content type and the Bearer token (Manager role).
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeToken(t, 999, 3)) 

	// Create a ResponseRecorder to capture the HTTP response.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 6. ASSERTIONS (STATUS CODE)
	// If the status is not 200, the middleware or handler blocked the request incorrectly.
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	// 7. ASSERTIONS (DATABASE INTEGRITY)
	// Fetch the record directly from the database to ensure persistence.
	var updated model.Course
	if err := db.DB.First(&updated, course.ID).Error; err != nil {
		t.Fatalf("failed to fetch updated course: %v", err)
	}
	
	// Compare actual DB values against the input payload.
	if updated.CourseName != "New Name" || updated.CourseCode != "NEW-001" || updated.Capacity != 15 {
		t.Fatalf("course not updated as expected: %+v", updated)
	}
}

/**
 * TestStudentCannotUpdateClass verifies the "Negative Path" for unauthorized roles.
 * * Expected Outcome:
 * A user with Role ID 1 (Student) should receive a 403 Forbidden status code.
 * Students should only have Read-Only access to the class catalog.
 */
func TestStudentCannotUpdateClass(t *testing.T) {
	// 1. SETUP
	setupManagerRouteTestDB(t)

	// 2. SEEDING
	seedRole(t, 1, "Student")
	course := seedCourseForUpdate(t)
	r := routes.SetupRouter()

	// 3. PAYLOAD
	body := map[string]any{
		"name": "Should Fail",
		"capacity": 15,
	}
	b, _ := json.Marshal(body)

	// 4. EXECUTION
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/classes/%d", course.ID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// Note the role_id=1 in the token generation below.
	req.Header.Set("Authorization", "Bearer "+makeToken(t, 1000, 1)) 

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 5. ASSERTION
	// The response should be 403 Forbidden because the user lacks 'Manager' privileges.
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d, body=%s", w.Code, w.Body.String())
	}
}

/*
SECTION 3: DATABASE HELPERS & DATA SEEDING
These functions manage the lifecycle of the test database and provide structured
ways to insert complex data relationships (Courses + Sessions).
*/

/**
 * setupManagerTestDBWithSessions initializes a temporary SQLite database.
 * * Why SQLite? 
 * It allows for fast, isolated tests without requiring a persistent Postgres/MySQL instance.
 * It runs in-memory, meaning the data is wiped automatically after the test finishes.
 */
func setupManagerTestDBWithSessions(t *testing.T) {
	t.Helper()
	// Disable Gin logging to keep test console output clean.
	gin.SetMode(gin.TestMode)

	// Create a unique Data Source Name (DSN) using UnixNano to avoid conflicts 
	// between parallel test runs.
	dsn := fmt.Sprintf("file:mgr_session_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	
	// Open the GORM connection.
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Enable Foreign Key constraints (SQLite disables them by default).
	// This ensures that relationships (like CourseID in ClassSession) are validated.
	if err := testDB.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Run AutoMigrate to create the table structure based on Go structs.
	// The order typically doesn't matter for GORM, but we list the core entities first.
	if err := testDB.AutoMigrate(
		&model.Role{},               // RBAC roles
		&model.User{},               // User credentials
		&model.UserInfo{},           // Profile details
		&model.Course{},             // Course metadata
		&model.ClassSession{},       // Specific instances of a course
		&model.Enrollment{},         // Student-Course mapping
		&model.UserDailyActivity{},  // Analytics/Logs
		&model.ManagerInviteCode{},  // Registration tokens
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// Assign the test DB instance to the global database pointer used by the app.
	db.DB = testDB
}

/**
 * seedManagerCourseWithSession creates a parent Course and a child ClassSession.
 * * This helper is essential for testing "Class Management" features where a 
 * course is just a template, and sessions are the actual events students attend.
 *
 * @param t        *testing.T - Testing context.
 * @param name     string     - Name of the course.
 * @param capacity int        - Maximum number of participants.
 * @return model.Course       - The newly created course object.
 */
func seedManagerCourseWithSession(t *testing.T, name string, capacity int) model.Course {
	t.Helper()

	// Parse string times into model-compatible formats.
	startTime, _ := model.ParseTimeOnly("09:00")
	endTime, _ := model.ParseTimeOnly("10:00")

	// Instantiate the Course model.
	course := model.Course{
		CourseName: name,
		// Using UnixNano for CourseCode to ensure uniqueness across seeds.
		CourseCode: fmt.Sprintf("MGR-%d", time.Now().UnixNano()),
		Capacity:   capacity,
		Category:   "Fitness",
		StartTime:  startTime,
		EndTime:    endTime,
		Weekday:    "Monday",
		Duration:   60,
	}

	// Persist the course to the database.
	if err := db.DB.Create(&course).Error; err != nil {
		t.Fatalf("failed to seed course: %v", err)
	}

	// Create a specific session date (tomorrow).
	sessionDate := time.Now().AddDate(0, 0, 1)

	// Instantiate the ClassSession model, linking it to the course via ID.
	session := model.ClassSession{
		CourseID:    course.ID,
		SessionDate: sessionDate.Format("2006-01-02"),
		// Construct specific UTC timestamps for start and end times.
		StartAt:     time.Date(sessionDate.Year(), sessionDate.Month(), sessionDate.Day(), 9, 0, 0, 0, time.UTC),
		EndAt:       time.Date(sessionDate.Year(), sessionDate.Month(), sessionDate.Day(), 10, 0, 0, 0, time.UTC),
		Status:      "scheduled",
		Capacity:    capacity,
	}

	// Persist the session. Failure here likely means a foreign key violation.
	if err := db.DB.Create(&session).Error; err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}

	return course
}

/* END OF TEST SUITE 
Summary of Logic:


/*
|--------------------------------------------------------------------------
| SEEDER UTILITIES: ENROLLMENT LOGIC
|--------------------------------------------------------------------------
| These functions handle the "Seed" phase of the AAA (Arrange-Act-Assert) 
| testing pattern. They ensure the database state is correctly prepared 
| before the actual HTTP request is executed.
*/

/**
 * seedManagerEnrollment creates a many-to-many relationship record between a 
 * User and a Course, specifically targeting an active ClassSession.
 *
 * @param t        *testing.T - Pointer to the test runner context.
 * @param userID   uint       - The unique ID of the student user.
 * @param courseID uint       - The primary key of the course to enroll in.
 * @return model.Enrollment   - The persisted enrollment object for further assertions.
 *
 * Technical Workflow:
 * 1. Dependency Lookup: An enrollment requires a specific session. We query
 * the first "scheduled" session for the given courseID, sorted by date.
 * 2. Foreign Key Integrity: Ensures that the SessionID exists in the db
 * to prevent database constraint failures during the create operation.
 * 3. Status Mapping: Uses the internal model constant 'EnrollmentStatusEnrolled'.
 */
func seedManagerEnrollment(t *testing.T, userID, courseID uint) model.Enrollment {
	// Mark function as helper so failure line numbers point back to the caller.
	t.Helper()

	var session model.ClassSession
	// Query Strategy: Find the earliest upcoming session for this course.
	// This simulates a real enrollment where a user signs up for the next available class.
	if err := db.DB.Where("course_id = ? AND status = ?", courseID, "scheduled").
		Order("session_date ASC").First(&session).Error; err != nil {
		t.Fatalf("failed to find valid session for enrollment: %v", err)
	}

	// Initialize the Enrollment struct.
	// Note: SessionID is a pointer (*uint) to support optional/null sessions if needed.
	enrollment := model.Enrollment{
		UserID:     userID,
		CourseID:   courseID,
		SessionID:  &session.ID,
		Status:     model.EnrollmentStatusEnrolled, // Standard active status
		EnrollTime: time.Now(),                     // Record the exact time of transaction
	}

	// Perform GORM Create operation.
	// Failure here usually implies a unique constraint violation (e.g., user already enrolled).
	if err := db.DB.Create(&enrollment).Error; err != nil {
		t.Fatalf("failed to seed enrollment record into database: %v", err)
	}
	
	return enrollment
}

/*
|--------------------------------------------------------------------------
| ENDPOINT TEST: GET /manager/users
|--------------------------------------------------------------------------
| This suite validates the User Management administrative endpoint.
| Access is restricted to users with Role ID 3 (Manager).
*/

/**
 * TestManagerListUsers_OK verifies that an authorized manager can retrieve
 * a paginated list of all users registered in the system.
 *
 * Validation Points:
 * - Middleware: Verifies the JWT is correctly parsed and RoleID 3 is accepted.
 * - Logic: Verifies that the pagination parameters (page/limit) are respected.
 * - Response: Verifies the JSON structure matches the expected API contract.
 */
func TestManagerListUsers_OK(t *testing.T) {
	// 1. SETUP: Initialize clean in-memory database with required schema.
	setupManagerTestDBWithSessions(t)

	// 2. DATA PREPARATION: Seed the RBAC roles.
	seedRole(t, 3, "Manager")
	seedRole(t, 1, "Student")

	// 3. BULK DATA CREATION: Generate dummy user records.
	// We create exactly 3 users to test the 'Total' and 'Length' assertions.
	for i := 0; i < 3; i++ {
		db.DB.Create(&model.User{
			Name:     fmt.Sprintf("User %d", i),
			// Use UnixNano to prevent email uniqueness collisions in high-speed tests.
			Email:    fmt.Sprintf("user-%d-%d@example.com", i, time.Now().UnixNano()),
			Password: "pass",
			RoleID:   1, // Assign everyone as a Student
		})
	}

	// 4. AUTHENTICATION: Generate a JWT with Manager privileges (role_id 3).
	// Subject ID 999 is used for the manager user in this context.
	token := makeToken(t, 999, 3) 
	r := routes.SetupRouter()

	// 5. REQUEST SIMULATION: Define the GET request with pagination query params.
	// Query Params: page=1 (starting page), limit=10 (items per page).
	req := httptest.NewRequest(http.MethodGet, "/manager/users?page=1&limit=10", nil)
	// Inject the Bearer token into the Request Header.
	req.Header.Set("Authorization", "Bearer "+token)
	
	// Prepare the recorder to capture the response.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 6. ASSERTION: HTTP Status Code.
	// Expected: 200 OK because the user has the correct 'Manager' role.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 7. ASSERTION: JSON Schema & Content.
	// Define a local anonymous struct to mirror the expected JSON response.
	var resp struct {
		Users      []model.User `json:"users"`
		Total      int64        `json:"total"`
		Page       int          `json:"page"`
		Limit      int          `json:"limit"`
		TotalPages int          `json:"total_pages"`
	}

	// Unmarshal the byte buffer into our response struct.
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode manager user list response: %v", err)
	}

	// Verify the pagination count (Total vs actual Slice length).
	if resp.Total != 3 {
		t.Fatalf("database total count mismatch: expected 3, got %d", resp.Total)
	}
	if len(resp.Users) != 3 {
		t.Fatalf("result set size mismatch: expected 3 users in slice, got %d", len(resp.Users))
	}
}

/**
 * TestManagerListUsers_Forbidden_NonManager verifies security enforcement.
 * * Case: A Student (RoleID 1) attempts to access the admin user list.
 * Expected Outcome: 403 Forbidden.
 * Why: Sensitive user data must not be exposed to non-administrative roles.
 */
func TestManagerListUsers_Forbidden_NonManager(t *testing.T) {
	// 1. SETUP: Fresh database environment.
	setupManagerTestDBWithSessions(t)
	seedRole(t, 1, "Student")

	// 2. AUTHENTICATION: Generate a JWT with Student privileges (role_id 1).
	token := makeToken(t, 999, 1) 
	r := routes.SetupRouter()

	// 3. REQUEST: Target the protected /manager endpoint.
	req := httptest.NewRequest(http.MethodGet, "/manager/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 4. ASSERTION: Verify the Authorization middleware blocked the request.
	// A status code of 403 indicates the token was valid but the role was insufficient.
	if w.Code != http.StatusForbidden {
		t.Fatalf("security breach: expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
	}
}

/*
|--------------------------------------------------------------------------
| ARCHITECTURAL NOTE ON PAGINATION
|--------------------------------------------------------------------------
| The 'ManagerListUsers' implementation uses Scopes or Offset/Limit logic.
| In these tests, we verify that the metadata (Page, Limit, TotalPages)
| matches the input query parameters and the actual database state.
*/

// ─── POST /manager/users/:id/enrollments ─────────────────────────

// func TestManagerAddUserEnrollment_OK(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&user)

// 	course := seedManagerCourseWithSession(t, "Yoga", 10)
// 	token := makeToken(t, 999, 3) // manager
// 	r := routes.SetupRouter()

// 	body, _ := json.Marshal(map[string]uint{"course_id": course.ID})
// 	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/manager/users/%d/enrollments", user.ID), bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusCreated {
// 		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
// 	}

// 	// Verify enrollment has session_id set
// 	var enrollment model.Enrollment
// 	if err := db.DB.Where("user_id = ? AND course_id = ?", user.ID, course.ID).First(&enrollment).Error; err != nil {
// 		t.Fatalf("enrollment not found in DB: %v", err)
// 	}
// 	if enrollment.SessionID == nil {
// 		t.Fatal("expected session_id to be set, got nil")
// 	}
// 	if enrollment.Status != model.EnrollmentStatusEnrolled {
// 		t.Fatalf("expected status 'enrolled', got '%s'", enrollment.Status)
// 	}
// }

// func TestManagerAddUserEnrollment_Duplicate(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&user)

// 	course := seedManagerCourseWithSession(t, "Spin", 10)
// 	seedManagerEnrollment(t, user.ID, course.ID)

// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	body, _ := json.Marshal(map[string]uint{"course_id": course.ID})
// 	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/manager/users/%d/enrollments", user.ID), bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusConflict {
// 		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
// 	}
// }

// func TestManagerAddUserEnrollment_ClassFull(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	course := seedManagerCourseWithSession(t, "Full Class", 1) // capacity=1

// 	// Fill the spot
// 	existing := model.User{Name: "Existing", Email: fmt.Sprintf("e-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&existing)
// 	seedManagerEnrollment(t, existing.ID, course.ID)

// 	// Try to add another
// 	newUser := model.User{Name: "New User", Email: fmt.Sprintf("n-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&newUser)

// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	body, _ := json.Marshal(map[string]uint{"course_id": course.ID})
// 	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/manager/users/%d/enrollments", newUser.ID), bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusConflict {
// 		t.Fatalf("expected 409 (class is full), got %d: %s", w.Code, w.Body.String())
// 	}
// }

// func TestManagerAddUserEnrollment_UserNotFound(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")

// 	course := seedManagerCourseWithSession(t, "Yoga", 10)
// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	body, _ := json.Marshal(map[string]uint{"course_id": course.ID})
// 	req := httptest.NewRequest(http.MethodPost, "/manager/users/9999/enrollments", bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusNotFound {
// 		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
// 	}
// }

// ─── DELETE /manager/users/:id/enrollments/:course_id ────────────

// func TestManagerDeleteUserEnrollment_OK(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&user)

// 	course := seedManagerCourseWithSession(t, "Rowing", 10)
// 	seedManagerEnrollment(t, user.ID, course.ID)

// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	path := fmt.Sprintf("/manager/users/%d/enrollments/%d", user.ID, course.ID)
// 	req := httptest.NewRequest(http.MethodDelete, path, nil)
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
// 	}

// 	// Verify deleted
// 	var count int64
// 	db.DB.Model(&model.Enrollment{}).Where("user_id = ? AND course_id = ?", user.ID, course.ID).Count(&count)
// 	if count != 0 {
// 		t.Fatalf("expected enrollment to be deleted, but found %d", count)
// 	}
// }

// func TestManagerDeleteUserEnrollment_NotFound(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&user)

// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/manager/users/%d/enrollments/9999", user.ID), nil)
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusNotFound {
// 		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
// 	}
// }

// // ─── GET /manager/users/:id/enrollments ──────────────────────────

/*
Package manager_test - Module: Enrollment Management API

This test suite validates the administrative capabilities provided to users with the 
"Manager" role. Specifically, it tests the endpoints that allow managers to oversee 
student activity across the platform.

KEY CONCEPTS TESTED:
1.  Role-Based Access Control (RBAC): Ensuring only tokens with RoleID 3 (Manager) 
    can access student data.
2.  Route Parameter Binding: Verifying that the {userID} from the URL correctly 
    filters the enrollment database.
3.  JSON Serialization: Confirming that the API response strictly follows the 
    defined Enrollment model schema.
4.  Error Handling: Proper HTTP 404 responses when querying non-existent entities.
*/

// =============================================================================
// TEST CASE: TestManagerListUserEnrollments_OK
// =============================================================================
// Description:
//   Tests the successful retrieval of all class enrollments for a specific student 
//   by a user with Manager privileges.
//
// Setup Workflow:
//   1. Initialize a clean Manager-context database.
//   2. Seed essential roles (Manager and Student).
//   3. Create a mock student user.
//   4. Generate two distinct courses, each with an active session.
//   5. Enroll the student in both courses to create a testable dataset.
//
// Validation:
//   - HTTP Status must be 200 (OK).
//   - The JSON body must contain exactly 2 enrollment records.
// =============================================================================
func TestManagerListUserEnrollments_OK(t *testing.T) {
	// Initialize the isolated test environment.
	setupManagerTestDBWithSessions(t)

	// --- STEP 1: PREREQUISITE DATA SEEDING ---
	// Seed roles to satisfy foreign key constraints. 
	// Role 3 = Manager, Role 1 = Student.
	seedRole(t, 3, "Manager")
	seedRole(t, 1, "Student")

	// Create the target student user.
	// We use UnixNano to ensure email uniqueness in parallel test runs.
	user := model.User{
		Name:     "Test User", 
		Email:    fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), 
		Password: "pass", 
		RoleID:   1,
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to seed student user: %v", err)
	}

	// --- STEP 2: COURSE & ENROLLMENT GENERATION ---
	// Create two courses with associated class sessions.
	course1 := seedManagerCourseWithSession(t, "Course A", 10)
	course2 := seedManagerCourseWithSession(t, "Course B", 10)

	// Perform the enrollment operations in the database.
	seedManagerEnrollment(t, user.ID, course1.ID)
	seedManagerEnrollment(t, user.ID, course2.ID)

	// --- STEP 3: HTTP REQUEST SIMULATION ---
	// Generate a JWT token for a Manager (ID 999, Role 3).
	token := makeToken(t, 999, 3)
	
	// Initialize the Gin router with manager-protected routes.
	r := routes.SetupRouter()

	// Construct the GET request targeting the specific student's ID.
	endpoint := fmt.Sprintf("/manager/users/%d/enrollments", user.ID)
	req := httptest.NewRequest(http.MethodGet, endpoint, nil)
	
	// Set the Bearer token for authentication middleware.
	req.Header.Set("Authorization", "Bearer "+token)
	
	// Create a recorder to capture the HTTP response.
	w := httptest.NewRecorder()
	
	// Dispatch the request.
	r.ServeHTTP(w, req)

	// --- STEP 4: ASSERTIONS ---
	// Verify the HTTP response status code.
	if w.Code != http.StatusOK {
		t.Fatalf("API failure: expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Define a temporary struct to decode the paginated or wrapped response.
	var resp struct {
		Enrollments []model.Enrollment `json:"enrollments"`
	}
	
	// Unmarshal the JSON response body.
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("serialization error: failed to decode JSON response: %v", err)
	}

	// Validate the number of enrollments returned matches our seeded data.
	if len(resp.Enrollments) != 2 {
		t.Fatalf("data integrity error: expected 2 enrollments in list, but found %d", len(resp.Enrollments))
	}
}

// =============================================================================
// TEST CASE: TestManagerListUserEnrollments_UserNotFound
// =============================================================================
// Description:
//   Validates the system's resilience when a Manager queries an invalid User ID.
//
// Expected Outcome:
//   - The system should recognize that User ID 9999 does not exist.
//   - The system should return a 404 Not Found status rather than an empty list.
// =============================================================================
func TestManagerListUserEnrollments_UserNotFound(t *testing.T) {
	// Initialize the test database.
	setupManagerTestDBWithSessions(t)
	
	// Seed the Manager role so the token authentication succeeds.
	seedRole(t, 3, "Manager")

	// Create a valid Manager token.
	token := makeToken(t, 999, 3)
	r := routes.SetupRouter()

	// Construct a request for a non-existent student (ID 9999).
	req := httptest.NewRequest(http.MethodGet, "/manager/users/9999/enrollments", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// --- ASSERTIONS ---
	// The API must return 404 to indicate the primary resource (the User) was not found.
	if w.Code != http.StatusNotFound {
		t.Fatalf("boundary error: expected 404 Not Found for invalid user, got %d: %s", w.Code, w.Body.String())
	}
}

// -----------------------------------------------------------------------------
// TECHNICAL NOTE ON MIDDLEWARE:
// Both tests rely on the 'AuthMiddleware' and 'RoleMiddleware'.
// If the token generation or role seeding fails, these tests will return 401/403.
// -----------------------------------------------------------------------------
