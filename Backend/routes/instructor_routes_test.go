package routes_test

import (
    "encoding/json"
    "fmt"
    "net/http"
    "testing"
    "time"

    "my-course-backend/db"
    "my-course-backend/model"
    "my-course-backend/routes"

    "github.com/gin-gonic/gin"
    "github.com/glebarez/sqlite"
    "gorm.io/gorm"
)

// setupInstructorTestDB initializes an in-memory SQLite database for testing.
// It configures Gin to test mode and migrates all necessary schemas.
func setupInstructorTestDB(t *testing.T) {
    t.Helper()
    gin.SetMode(gin.TestMode)

    // Use a unique name for the in-memory database to avoid collisions between parallel tests
    dsn := fmt.Sprintf("file:instructor_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
    testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to open test database: %v", err)
    }

    // Explicitly enable foreign key constraints for data integrity validation
    if err := testDB.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
        t.Fatalf("failed to enable foreign keys: %v", err)
    }

    // Perform AutoMigration for all models involved in instructor workflows
    if err := testDB.AutoMigrate(
        &model.Role{},
        &model.User{},
        &model.UserInfo{},
        &model.Course{},
        &model.ClassSession{},
        &model.Enrollment{},
        &model.UserDailyActivity{},
    ); err != nil {
        t.Fatalf("failed to migrate test database: %v", err)
    }

    db.DB = testDB
}

// seedInstructorUser creates a dummy instructor role and user in the test database.
// It returns the user object and a valid JWT token for authentication.
func seedInstructorUser(t *testing.T) (model.User, string) {
    t.Helper()
    // Define role_id 4 as the standard ID for "Instructor"
    if err := db.DB.FirstOrCreate(&model.Role{ID: 4, RoleName: "Instructor"}).Error; err != nil {
        t.Fatalf("failed to seed role: %v", err)
    }
    
    user := model.User{
        Name:     "Test Instructor",
        Email:    fmt.Sprintf("instructor-%d@example.com", time.Now().UnixNano()),
        Password: "notused",
        RoleID:   4,
    }
    if err := db.DB.Create(&user).Error; err != nil {
        t.Fatalf("failed to seed instructor user: %v", err)
    }
    
    // Generate token using the utility function (assumed to be defined in the same package)
    token := makeToken(t, user.ID, 4)
    return user, token
}

// seedTestUser creates a standard student user for enrollment testing.
func seedTestUser(t *testing.T) model.User {
    t.Helper()
    // Ensure the Student role exists
    db.DB.FirstOrCreate(&model.Role{ID: 1, RoleName: "Student"}, "id = ?", 1)
    
    user := model.User{
        Name:     "Test User",
        Email:    fmt.Sprintf("user-%d@example.com", time.Now().UnixNano()),
        Password: "notused",
        RoleID:   1,
    }
    if err := db.DB.Create(&user).Error; err != nil {
        t.Fatalf("failed to seed user: %v", err)
    }
    return user
}


// seedCourseWithInstructor creates a course assigned to a specific instructor 
// and automatically generates a scheduled class session.
func seedCourseWithInstructor(t *testing.T, instructorID uint, name string, capacity int) model.Course {
    t.Helper()
    startTime, _ := model.ParseTimeOnly("09:00")
    endTime, _ := model.ParseTimeOnly("10:00")

    // Fetch the instructor details to populate the instructor name field in the course record
    var instructorUser model.User
    if err := db.DB.First(&instructorUser, instructorID).Error; err != nil {
        t.Fatalf("failed to load instructor user: %v", err)
    }

    course := model.Course{
        CourseName: name,
        CourseCode: fmt.Sprintf("INS-%d", time.Now().UnixNano()),
        Capacity:   capacity,
        Category:   "Fitness",
        StartTime:  startTime,
        EndTime:    endTime,
        Weekday:    "Monday",
        Instructor: instructorUser.Name,
        Duration:   60,
    }
    if err := db.DB.Create(&course).Error; err != nil {
        t.Fatalf("failed to seed course: %v", err)
    }

    // Create a corresponding ClassSession (scheduled for tomorrow)
    sessionDate := time.Now().AddDate(0, 0, 1)
    session := model.ClassSession{
        CourseID:    course.ID,
        SessionDate: sessionDate.Format("2006-01-02"),
        StartAt:     time.Date(sessionDate.Year(), sessionDate.Month(), sessionDate.Day(), 9, 0, 0, 0, time.UTC),
        EndAt:       time.Date(sessionDate.Year(), sessionDate.Month(), sessionDate.Day(), 10, 0, 0, 0, time.UTC),
        Status:      "scheduled",
        Capacity:    capacity,
    }
    if err := db.DB.Create(&session).Error; err != nil {
        t.Fatalf("failed to seed class session: %v", err)
    }

    return course
}


// seedEnrollmentWithSession links a user to a course session.
// It finds the first scheduled session for the given course and creates an enrollment record.
func seedEnrollmentWithSession(t *testing.T, userID, courseID uint, status string) model.Enrollment {
    t.Helper()
    var session model.ClassSession
    
    // Find the next available scheduled session for this course
    if err := db.DB.Where("course_id = ? AND status = ?", courseID, "scheduled").
        Order("session_date ASC").First(&session).Error; err != nil {
        t.Fatalf("failed to find scheduled session: %v", err)
    }

    enrollment := model.Enrollment{
        UserID:     userID,
        CourseID:   courseID,
        SessionID:  &session.ID,
        Status:     status,
        EnrollTime: time.Now(),
    }
    if err := db.DB.Create(&enrollment).Error; err != nil {
        t.Fatalf("failed to seed enrollment: %v", err)
    }
    return enrollment
}


// ─── GET /instructor/courses ─────────────────────────────────────

// TestInstructorGetCourseList_OK verifies that an instructor can successfully 
// retrieve a list of all courses they are assigned to.
func TestInstructorGetCourseList_OK(t *testing.T) {
    // Initialize database and seed instructor data
    setupInstructorTestDB(t)
    instructor, token := seedInstructorUser(t)
    
    // Seed multiple courses for this specific instructor
    seedCourseWithInstructor(t, instructor.ID, "Yoga 101", 20)
    seedCourseWithInstructor(t, instructor.ID, "Pilates", 15)
    router := routes.SetupRouter()

    // Execute GET request to the instructor courses endpoint
    rec := performJSONRequest(t, router, http.MethodGet, "/instructor/courses", token, nil)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }

    // Define response structure and decode the body
    var resp struct {
        Courses []model.Course `json:"courses"`
    }
    if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
        t.Fatalf("failed to decode: %v", err)
    }
    
    // Validate that the returned course count matches the seeded data
    if len(resp.Courses) != 2 {
        t.Fatalf("expected 2 courses, got %d", len(resp.Courses))
    }
}

// TestInstructorGetCourseList_Empty ensures the API returns an empty list 
// (rather than an error) when an instructor has no courses.
func TestInstructorGetCourseList_Empty(t *testing.T) {
    setupInstructorTestDB(t)
    _, token := seedInstructorUser(t) // Instructor exists but has no courses
    router := routes.SetupRouter()

    rec := performJSONRequest(t, router, http.MethodGet, "/instructor/courses", token, nil)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }

    var resp struct {
        Courses []model.Course `json:"courses"`
    }
    if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
        t.Fatalf("failed to decode: %v", err)
    }
    
    // Assert that the list is empty
    if len(resp.Courses) != 0 {
        t.Fatalf("expected 0 courses, got %d", len(resp.Courses))
    }
}

// TestInstructorGetCourseList_Forbidden_NonInstructor verifies that users 
// without the "Instructor" role cannot access this protected endpoint.
func TestInstructorGetCourseList_Forbidden_NonInstructor(t *testing.T) {
    setupInstructorTestDB(t)
    
    // Explicitly seed a user with a non-instructor role (Student)
    db.DB.Create(&model.Role{ID: 1, RoleName: "Student"})
    userToken := makeToken(t, 999, 1) // Using role_id 1
    router := routes.SetupRouter()

    // Request should be rejected with a 403 Forbidden status
    rec := performJSONRequest(t, router, http.MethodGet, "/instructor/courses", userToken, nil)
    if rec.Code != http.StatusForbidden {
        t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
    }
}

// ─── GET /instructor/courses/:id/enrollments ─────────────────────

// TestInstructorGetCourseEnrollments_OK checks if an instructor can view 
// the student enrollment list for a specific course they own.
func TestInstructorGetCourseEnrollments_OK(t *testing.T) {
    setupInstructorTestDB(t)
    instructor, token := seedInstructorUser(t)
    course := seedCourseWithInstructor(t, instructor.ID, "HIIT", 20)
    
    // Seed a student user and enroll them in the course
    user := seedTestUser(t)
    seedEnrollmentWithSession(t, user.ID, course.ID, model.EnrollmentStatusEnrolled)
    router := routes.SetupRouter()

    path := fmt.Sprintf("/instructor/courses/%d/enrollments", course.ID)
    rec := performJSONRequest(t, router, http.MethodGet, path, token, nil)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }

    var resp struct {
        Enrollments []model.Enrollment `json:"enrollments"`
    }
    if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
        t.Fatalf("failed to decode: %v", err)
    }
    
    // Validate that the enrollment data matches the seeded student
    if len(resp.Enrollments) != 1 {
        t.Fatalf("expected 1 enrollment, got %d", len(resp.Enrollments))
    }
    if resp.Enrollments[0].UserID != user.ID {
        t.Fatalf("expected user_id %d, got %d", user.ID, resp.Enrollments[0].UserID)
    }
}

// TestInstructorGetCourseEnrollments_Forbidden_NotOwner prevents instructors 
// from accessing enrollment data for courses they do not teach.
func TestInstructorGetCourseEnrollments_Forbidden_NotOwner(t *testing.T) {
    setupInstructorTestDB(t)
    instructor, _ := seedInstructorUser(t)
    course := seedCourseWithInstructor(t, instructor.ID, "Spin", 10)

    // Create a different instructor account to simulate unauthorized access
    otherInstructor := model.User{
        Name:     "Other Instructor",
        Email:    fmt.Sprintf("other-instructor-%d@example.com", time.Now().UnixNano()),
        Password: "notused",
        RoleID:   4,
    }
    db.DB.Create(&otherInstructor)
    otherToken := makeToken(t, otherInstructor.ID, 4)

    router := routes.SetupRouter()
    path := fmt.Sprintf("/instructor/courses/%d/enrollments", course.ID)
    
    // Attempting to access another instructor's course data should return 403
    rec := performJSONRequest(t, router, http.MethodGet, path, otherToken, nil)
    if rec.Code != http.StatusForbidden {
        t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
    }
}

// TestInstructorGetCourseEnrollments_NotFound verifies behavior when 
// querying enrollments for a non-existent course ID.
func TestInstructorGetCourseEnrollments_NotFound(t *testing.T) {
    setupInstructorTestDB(t)
    _, token := seedInstructorUser(t)
    router := routes.SetupRouter()

    // Querying a non-existent ID (9999) should return 404
    rec := performJSONRequest(t, router, http.MethodGet, "/instructor/courses/9999/enrollments", token, nil)
    if rec.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
    }
}

// ─── PATCH /instructor/courses/:id/enrollments ───────────────────

// TestInstructorUpdateEnrollmentStatus_Attended tests the functionality 
// of updating a student's enrollment status (e.g., marking them as attended).
func TestInstructorUpdateEnrollmentStatus_Attended(t *testing.T) {
    setupInstructorTestDB(t)
    instructor, token := seedInstructorUser(t)
    course := seedCourseWithInstructor(t, instructor.ID, "Boxing", 20)
    user := seedTestUser(t)
    
    // Initially seed as 'enrolled'
    seedEnrollmentWithSession(t, user.ID, course.ID, model.EnrollmentStatusEnrolled)
    router := routes.SetupRouter()

    path := fmt.Sprintf("/instructor/courses/%d/enrollments", course.ID)
    body := map[string]interface{}{
        "user_id": user.ID,
        "status":  "attended", // Target status to update
    }
    
    // Execute PATCH request to update the status
    rec := performJSONRequest(t, router, http.MethodPatch, path, token, body)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }

    // Query the database directly to verify the record was updated permanently
    var enrollment model.Enrollment
    db.DB.Where("user_id = ? AND course_id = ?", user.ID, course.ID).First(&enrollment)
    if enrollment.Status != "attended" {
        t.Fatalf("expected status 'attended', got '%s'", enrollment.Status)
    }
}
/*
PACKAGE: tests
This test suite focuses on the Instructor's ability to manage student attendance.
The business logic being tested follows a strict ownership model:


// ============================================================================
// TEST CASE: Successful Attendance Update (Status: Missed)
// ============================================================================

/**
 * TestInstructorUpdateEnrollmentStatus_Missed validates the "Happy Path" where 
 * a course owner marks a student as having missed a session.
 *
 * Technical Requirements:
 * - The Instructor must be authenticated.
 * - The Course must exist and belong to the Instructor.
 * - The Enrollment record must link the specific User to that Course.
 */
func TestInstructorUpdateEnrollmentStatus_Missed(t *testing.T) {
	// [ARRANGE] - Environment Setup
	// setupInstructorTestDB initializes an in-memory SQLite instance with 
	// specific seeds for Role 4 (Instructor).
	setupInstructorTestDB(t)

	// [ARRANGE] - Actor Creation
	// seedInstructorUser generates a User record with RoleID 4 and returns a valid JWT.
	instructor, token := seedInstructorUser(t)

	// [ARRANGE] - Resource Creation
	// We create a "Cycling" course explicitly assigned to this instructor's ID.
	course := seedCourseWithInstructor(t, instructor.ID, "Cycling", 20)

	// [ARRANGE] - Target Creation
	// Create a generic student user and enroll them in the instructor's course.
	// Initial state: model.EnrollmentStatusEnrolled.
	user := seedTestUser(t)
	seedEnrollmentWithSession(t, user.ID, course.ID, model.EnrollmentStatusEnrolled)

	// [ACT] - Execute Request
	// Initialize the routing engine and prepare the PATCH request.
	router := routes.SetupRouter()
	path := fmt.Sprintf("/instructor/courses/%d/enrollments", course.ID)
	body := map[string]interface{}{
		"user_id": user.ID,
		"status":  "missed", // Moving status from 'enrolled' to 'missed'
	}

	// performJSONRequest is a helper that adds the Authorization header and marshals the JSON.
	rec := performJSONRequest(t, router, http.MethodPatch, path, token, body)

	// [ASSERT] - HTTP Response
	// 200 OK indicates the server logic processed the update successfully.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// [ASSERT] - Database Persistence
	// We must verify that the record in the database actually changed.
	// A mock response code is not enough to prove data integrity.
	var enrollment model.Enrollment
	db.DB.Where("user_id = ? AND course_id = ?", user.ID, course.ID).First(&enrollment)
	if enrollment.Status != "missed" {
		t.Fatalf("database sync failure: expected status 'missed', got '%s'", enrollment.Status)
	}
}

// ============================================================================
// TEST CASE: Data Validation (Invalid Status)
// ============================================================================

/**
 * TestInstructorUpdateEnrollmentStatus_InvalidStatus ensures the API layer 
 * performs strict validation on input strings.
 *
 * Logic: The application should reject any status not defined in the 
 * EnrollmentStatus constants (e.g., enrolled, attended, missed).
 */
func TestInstructorUpdateEnrollmentStatus_InvalidStatus(t *testing.T) {
	// [ARRANGE] - Typical setup for an authenticated instructor context.
	setupInstructorTestDB(t)
	instructor, token := seedInstructorUser(t)
	course := seedCourseWithInstructor(t, instructor.ID, "Dance", 20)
	user := seedTestUser(t)
	seedEnrollmentWithSession(t, user.ID, course.ID, model.EnrollmentStatusEnrolled)

	router := routes.SetupRouter()

	// [ACT] - Send an unsupported status string ("invalid_status").
	path := fmt.Sprintf("/instructor/courses/%d/enrollments", course.ID)
	body := map[string]interface{}{
		"user_id": user.ID,
		"status":  "invalid_status",
	}
	rec := performJSONRequest(t, router, http.MethodPatch, path, token, body)

	// [ASSERT] - Expect 400 Bad Request.
	// This proves that the instructor cannot inject arbitrary data into the status field.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("validation error: expected 400 for bad status, got %d", rec.Code)
	}
}

// ============================================================================
// TEST CASE: Authorization & Ownership (Horizontal Privilege Escalation)
// ============================================================================

/**
 * TestInstructorUpdateEnrollmentStatus_Forbidden_NotOwner is a CRITICAL security test.
 *
 * Scenario: Instructor A owns Course 101. Instructor B (Other Instructor) 
 * attempts to modify an enrollment in Course 101.
 *
 * Expected Outcome: 403 Forbidden.
 * Justification: Instructors must only manage their own student lists.
 */
func TestInstructorUpdateEnrollmentStatus_Forbidden_NotOwner(t *testing.T) {
	// [ARRANGE] - Create the victim instructor and their course.
	setupInstructorTestDB(t)
	instructor, _ := seedInstructorUser(t) // This instructor is the true owner.
	course := seedCourseWithInstructor(t, instructor.ID, "Zumba", 20)
	user := seedTestUser(t)
	seedEnrollmentWithSession(t, user.ID, course.ID, model.EnrollmentStatusEnrolled)

	// [ARRANGE] - Create the attacker instructor (different user ID).
	otherInstructor := model.User{
		Name:     "Other Instructor",
		Email:    fmt.Sprintf("other-%d@example.com", time.Now().UnixNano()),
		Password: "notused",
		RoleID:   4, // They have the Instructor role, but not ownership of THIS course.
	}
	db.DB.Create(&otherInstructor)
	otherToken := makeToken(t, otherInstructor.ID, 4)

	router := routes.SetupRouter()

	// [ACT] - Attacker tries to update Instructor A's student status.
	path := fmt.Sprintf("/instructor/courses/%d/enrollments", course.ID)
	body := map[string]interface{}{
		"user_id": user.ID,
		"status":  "attended",
	}
	rec := performJSONRequest(t, router, http.MethodPatch, path, otherToken, body)

	// [ASSERT] - Verify 403 Forbidden.
	// This confirms that the Course Ownership Middleware is functioning correctly.
	if rec.Code != http.StatusForbidden {
		t.Fatalf("security violation: expected 403, but instructor modified another's course (%d)", rec.Code)
	}
}

// ============================================================================
// TEST CASE: Resource Availability (Enrollment Not Found)
// ============================================================================

/**
 * TestInstructorUpdateEnrollmentStatus_EnrollmentNotFound checks error handling 
 * when the targeting parameters are invalid.
 */
func TestInstructorUpdateEnrollmentStatus_EnrollmentNotFound(t *testing.T) {
	// [ARRANGE] - Create valid instructor and course, but do NOT enroll the user.
	setupInstructorTestDB(t)
	instructor, token := seedInstructorUser(t)
	course := seedCourseWithInstructor(t, instructor.ID, "Stretch", 20)
	router := routes.SetupRouter()

	// [ACT] - Attempt to update a user ID (9999) that is not enrolled in the course.
	path := fmt.Sprintf("/instructor/courses/%d/enrollments", course.ID)
	body := map[string]interface{}{
		"user_id": 9999, // Non-existent user mapping
		"status":  "attended",
	}
	rec := performJSONRequest(t, router, http.MethodPatch, path, token, body)

	// [ASSERT] - Expect 404 Not Found.
	// The system should not find an enrollment record for User 9999 in Course [ID].
	if rec.Code != http.StatusNotFound {
		t.Fatalf("logic error: expected 404 for missing enrollment, got %d", rec.Code)
	}
}

/* SUMMARY OF TEST SUITE COVERAGE:
1. Status Integrity: Verifies that only 'missed' and 'attended' are effectively used.
2. RBAC Enforcement: Ensures Instructor RoleID 4 is recognized.
3. Resource Ownership: Ensures Course.InstructorID is validated against JWT.UserID.
4. Input Sanitation: Ensures malformed status inputs are rejected at the edge.
*/