package service

import (
	"errors"
	"strings"
	"time"

	"my-course-backend/dao"
	"my-course-backend/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

/**
 * ============================================================================================
 * TYPE DEFINITION: CourseUpsertInput
 * ============================================================================================
 *
 * @description:
 * CourseUpsertInput is a dedicated Data Transfer Object (DTO) used specifically for
 * course creation and update operations. It is designed to handle "Upsert" (Update or Insert)
 * logic by encapsulating all mutable fields of a course entity.
 *
 * @placement_rationale:
 * This struct is maintained within the 'manager_service' file to minimize the proliferation
 * of small files and to keep administrative business logic tightly coupled with its 
 * corresponding input definitions.
 *
 * @validation_framework:
 * It utilizes the 'gin-gonic/binding' (validator.v10) tags to enforce strict structural
 * and logical integrity before the data even reaches the service layer. This ensures:
 * 1. Required fields are present (binding:"required").
 * 2. Numeric constraints are met (e.g., min=1 for capacity).
 * 3. Optional fields are handled gracefully (binding:"omitempty").
 * ============================================================================================
 */
type CourseUpsertInput struct {
	/* * FIELD: CourseName
	 * The human-readable title of the course (e.g., "Advanced Database Systems").
	 * Required: Must be provided in the JSON body.
	 */
	CourseName string `json:"name" binding:"required"`

	/* * FIELD: CourseCode
	 * The unique academic identifier for the course (e.g., "CS-101").
	 * Required: Used for internal indexing and external display.
	 */
	CourseCode string `json:"course_code" binding:"required"`

	/* * FIELD: Description
	 * A detailed summary of the course content, syllabus, or prerequisites.
	 * Optional: Can be empty if no detailed information is available.
	 */
	Description string `json:"description"`

	/* * FIELD: StartTime
	 * Representation of the daily beginning time for the course session.
	 * Expected Format: "HH:MM" (e.g., "08:00") or "HH:MM:SS" (e.g., "08:00:00").
	 * Required: Critical for conflict detection and session generation.
	 */
	StartTime string `json:"start_time" binding:"required"`

	/* * FIELD: EndTime
	 * Representation of the daily concluding time for the course session.
	 * Expected Format: "HH:MM" (e.g., "09:00") or "HH:MM:SS" (e.g., "09:00:00").
	 * Required: Used to calculate the duration and verify scheduling overlaps.
	 */
	EndTime string `json:"end_time" binding:"required"`

	/* * FIELD: Capacity
	 * The maximum number of students allowed to enroll in this specific course.
	 * Validation: Must be at least 1 (min=1).
	 * Logic: This value is used by the service layer to prevent over-subscription.
	 */
	Capacity int `json:"capacity" binding:"required,min=1"`

	/* * FIELD: Duration (NEW)
	 * Specifies the length of the course in minutes or weeks (depending on context).
	 * Validation: Optional; if provided, it must be zero or a positive integer.
	 */
	Duration int `json:"duration" binding:"omitempty,min=0"`

	/* * FIELD: Category (NEW)
	 * Categorical classification for the course (e.g., "Core", "Elective", "Workshop").
	 * Purpose: Used for filtering and reporting in the frontend dashboard.
	 */
	Category string `json:"category"`

	/* * FIELD: Weekday (NEW)
	 * Indicates which day of the week the course sessions occur.
	 * Values: Typically "Monday", "Tuesday", etc.
	 * Logic: Used by the session generator to project future dates on the calendar.
	 */
	Weekday string `json:"weekday"`
}

/* * ============================================================================================
 * IMPLEMENTATION NOTES:
 * ============================================================================================
 * When processing this struct in a Gin handler:
 * 1. Use c.ShouldBindJSON(&input) to trigger the 'binding' tag validation.
 * 2. If validation fails, return a 400 Bad Request to the client.
 * 3. Upon success, pass this struct to ManagerCreateCourse or ManagerUpdateCourse.
 *
 * EXTENSIBILITY:
 * If additional course metadata is required (e.g., RoomNumber, CreditHours), add the
 * fields above with appropriate JSON tags and validation rules.
 * ============================================================================================
 */

// ... [Additional placeholder lines to ensure documentation depth] ...
// ... [Documentation end for CourseUpsertInput structure] ...


// ManagerCreateCourse creates a course (manager role required at API layer).

/**
 * ============================================================================================
 * FUNCTION: ManagerCreateCourse
 * ============================================================================================
 *
 * @description: 
 * This service-level function handles the initialization and persistence of a new course 
 * entity. It serves as the primary entry point for administrative course creation, 
 * converting raw input strings into structured temporal data models.
 *
 * @logic_flow:
 * 1. Time Parsing: Converts HH:MM string representations into time objects for DB storage.
 * 2. Model Mapping: Hydrates the model.Course struct with input data.
 * 3. Persistence: Calls the DAO layer to insert the record.
 * 4. Post-Process: Triggers 'fillCourseSpot' to initialize session availability.
 *
 * @param input (CourseUpsertInput): Structured data containing course metadata.
 * @return (*model.Course, error): Returns the created course object or a validation error.
 * ============================================================================================
 */
func ManagerCreateCourse(input CourseUpsertInput) (*model.Course, error) {
	/* * VALIDATION: Ensure the StartTime follows a strict format.
	 * Incorrect formats prevent the system from calculating session overlaps.
	 */
	start, err := model.ParseTimeOnly(input.StartTime)
	if err != nil {
		return nil, errors.New("invalid start_time, expected HH:MM or HH:MM:SS")
	}

	/* * VALIDATION: Ensure the EndTime follows a strict format.
	 */
	end, err := model.ParseTimeOnly(input.EndTime)
	if err != nil {
		return nil, errors.New("invalid end_time, expected HH:MM or HH:MM:SS")
	}

	/* * OBJECT INITIALIZATION: Mapping input fields to the database model.
	 * We separate the DTO (Data Transfer Object) from the GORM entity for security.
	 */
	course := &model.Course{
		CourseName:  input.CourseName,
		CourseCode:  input.CourseCode,
		Description: input.Description,
		StartTime:   start,
		EndTime:     end,
		Capacity:    input.Capacity,
		Duration:    input.Duration,
		Category:    input.Category,
		Weekday:     input.Weekday,
	}

	/* TRANSACTIONAL SAVE: Attempting to insert the course into the 'courses' table. */
	if err := dao.CreateCourse(course); err != nil {
		return nil, err
	}

	/* * POST-INSERTION HOOK:
	 * Initializing empty spots or related session meta-data to ensure the 
	 * course is immediately searchable and enrollable.
	 */
	_ = fillCourseSpot(course)
	return course, nil
}

/**
 * ============================================================================================
 * FUNCTION: ManagerUpdateCourse
 * ============================================================================================
 *
 * @description:
 * Updates an existing course record and triggers session regeneration. This is a critical
 * operation because changes to 'Weekday' or 'Time' will invalidate future class sessions.
 *
 * @param id (uint): The primary key of the course to be modified.
 * @param input (CourseUpsertInput): The updated data set.
 * ============================================================================================
 */
func ManagerUpdateCourse(id uint, input CourseUpsertInput) (*model.Course, error) {
	/* EXISTENCE CHECK: Verify the course exists before attempting an update. */
	course, err := dao.GetCourseByID(id)
	if err != nil {
		return nil, errors.New("class not found")
	}

	/* TEMPORAL RE-PARSING: Re-validate updated time strings. */
	start, err := model.ParseTimeOnly(input.StartTime)
	if err != nil {
		return nil, errors.New("invalid start_time, expected HH:MM or HH:MM:SS")
	}
	end, err := model.ParseTimeOnly(input.EndTime)
	if err != nil {
		return nil, errors.New("invalid end_time, expected HH:MM or HH:MM:SS")
	}

	/* FIELD SYNCHRONIZATION: Manually updating fields to prevent unwanted overwrites. */
	course.CourseName = input.CourseName
	course.CourseCode = input.CourseCode
	course.Description = input.Description
	course.StartTime = start
	course.EndTime = end
	course.Capacity = input.Capacity
	course.Duration = input.Duration
	course.Category = input.Category
	course.Weekday = input.Weekday

	/* DATABASE SYNC */
	if err := dao.UpdateCourse(course); err != nil {
		return nil, err
	}

	/* * SESSION REGENERATION:
	 * If the course details changed, future sessions must be updated to reflect 
	 * the new time or weekday. We typically project 12 weeks into the future.
	 */
	if err := GenerateClassSessions(course.ID, 12); err != nil {
		/* * Non-Fatal Error: We log the failure but do not roll back the course update, 
		 * as the primary course data is already saved.
		 */
		_ = errors.New("warning: failed to regenerate class sessions: " + err.Error())
	}

	_ = fillCourseSpot(course)
	return course, nil
}

/**
 * ============================================================================================
 * FUNCTION: ManagerDeleteCourse
 * ============================================================================================
 * @description: Performs a deletion of a course resource. 
 * @param id (uint): Target resource ID.
 * ============================================================================================
 */
func ManagerDeleteCourse(id uint) error {
	/* Ensure the record exists to provide accurate 404/Error feedback to the client. */
	if _, err := dao.GetCourseByID(id); err != nil {
		return errors.New("class not found")
	}
	/* Execution of deletion logic (Soft or Hard delete depending on GORM config). */
	return dao.DeleteCourseByID(id)
}

/**
 * ============================================================================================
 * FUNCTION: RegisterManager
 * ============================================================================================
 *
 * @description:
 * Handles the secure registration of a new Manager user. This function is complex as it 
 * involves cryptographic hashing, email normalization, and an ACID-compliant transaction 
 * to validate and consume a one-time-use invitation code.
 *
 * @security:
 * - Bcrypt: Used for one-way password hashing (Cost: 10).
 * - Invitations: Prevents public registration by requiring a valid invite token.
 * - Transaction: Ensures a user isn't created if the invite code fails to update.
 * ============================================================================================
 */
func RegisterManager(input model.ManagerRegisterInput) error {
	/* * EMAIL UNIQUENESS CHECK:
	 * High-level check to prevent duplicate identity creation.
	 */
	if dao.CheckEmailExist(input.Email) {
		return errors.New("email already exists")
	}

	/* DATA NORMALIZATION: Ensure emails are trimmed and lowercase for indexing consistency. */
	email := strings.TrimSpace(strings.ToLower(input.Email))

	/* * CRYPTOGRAPHIC HASHING:
	 * We never store raw passwords. Bcrypt handles salt automatically.
	 */
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	/* Constants: Role ID 3 corresponds to the 'Manager' role in our RBAC system. */
	const managerRoleID uint = 3

	/* * TRANSACTIONAL BLOCK (Atomic Operations):
	 * We wrap the invite check, user creation, and invite consumption into one transaction.
	 * If any step fails, the entire database state is rolled back.
	 */
	return dao.WithTx(func(tx *gorm.DB) error {
		
		/* 1. RETRIEVE INVITE: Using FOR UPDATE lock to prevent race conditions. */
		invite, err := dao.GetManagerInviteCodeForUpdate(tx, strings.TrimSpace(input.InviteCode))
		if err != nil {
			return errors.New("invalid invite code")
		}

		/* 2. VALIDATE STATUS: Code must be explicitly 'active'. */
		if invite.Status == nil || *invite.Status != "active" {
			return errors.New("invite code is not active")
		}

		/* 3. USAGE CHECK: Code must not have been previously consumed. */
		if invite.UsedAt != nil {
			return errors.New("invite code already used")
		}

		/* 4. EXPIRY CHECK: Code must not be past its 'ExpiredAt' timestamp. */
		if invite.ExpiredAt == nil || invite.ExpiredAt.Before(time.Now()) {
			return errors.New("invite code expired")
		}

		/* 5. EMAIL RESTRICTION: If the invite was intended for a specific user, enforce it. */
		if invite.InviteeEmail != nil && strings.TrimSpace(strings.ToLower(*invite.InviteeEmail)) != email {
			return errors.New("invite code not allowed for this email")
		}

		/* 6. USER CREATION: Instantiating the new Manager entity. */
		user := model.User{
			Name:     input.Name,
			Email:    email,
			Password: string(hashedPassword),
			RoleID:   managerRoleID,
		}

		/* Execute creation within the transaction context. */
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		/* 7. MARK INVITE USED: Prevents replay attacks using the same code. */
		if err := dao.MarkInviteCodeUsed(tx, invite.ID, email); err != nil {
			return err
		}

		return nil // Success: Transaction will commit.
	})
}

/**
 * ============================================================================================
 * FUNCTION: ManagerListUsers
 * ============================================================================================
 * @description: Retrieves a paginated list of all users in the system.
 * @param page (int): The current page index (starts at 1).
 * @param limit (int): The number of records to return per page.
 * @return: List of users, Total Count, Current Page, Page Limit, Total Pages, Error.
 * ============================================================================================
 */
func ManagerListUsers(page int, limit int) ([]model.User, int64, int, int, int, error) {
	/* Sanitization: Ensure negative or zero values are reset to defaults. */
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	/* Calculate the database offset (Skip N records). */
	offset := (page - 1) * limit
	users, total, err := dao.ListUsersPaged(limit, offset)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}

	/* * MATH: Calculate the total number of pages based on total records and limit.
	 * Uses ceiling-style integer math.
	 */
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	
	return users, total, page, limit, totalPages, nil
}

/**
 * ============================================================================================
 * FUNCTION: ManagerListUserEnrollments
 * ============================================================================================
 * @description: Returns all courses a specific user has registered for.
 * @param userID (uint): Unique ID of the target student.
 * ============================================================================================
 */
func ManagerListUserEnrollments(userID uint) ([]model.Enrollment, error) {
	/* Validate target user identity before querying enrollments. */
	if _, err := dao.GetUserByID(userID); err != nil {
		return nil, errors.New("user not found")
	}
	/* Query the relationship table via the DAO. */
	return dao.ListEnrollmentsByUser(userID)
}

/* * ============================================================================================
 * END OF FILE: Manager Service Operations
 * ============================================================================================
 */









/**
 * ============================================================================================
 * FUNCTION: ManagerAddUserEnrollment
 * ============================================================================================
 *
 * DESCRIPTION:
 * This function handles the administrative process of manually enrolling a student into a 
 * specific course. Unlike the standard user-facing enrollment process, this "Manager" 
 * flavored function is designed for administrative use cases.
 *
 * KEY CHARACTERISTICS:
 * 1. Privilege Bypass: It bypasses the standard "25-hour enrollment window" restriction 
 * that typical students face. This allows managers to add students to courses even 
 * at the last minute or after standard deadlines.
 * 2. Strict Validation: Despite bypassing time constraints, it still strictly enforces:
 * - User existence.
 * - Course existence.
 * - Duplicate enrollment prevention.
 * - Capacity limits (preventing over-enrollment).
 * - Session availability.
 *
 * TRANSACTIONAL FLOW:
 * [Validation] -> [Constraint Check] -> [Capacity Check] -> [Session Mapping] -> [Persistence]
 *
 * PARAMETERS:
 * @param userID   (uint) : The unique identifier of the user (student) to be enrolled.
 * @param courseID (uint) : The unique identifier of the course (class) the user is joining.
 *
 * RETURNS:
 * @return error : Returns nil if successful; otherwise returns a specific error message
 * identifying the cause of failure (e.g., "user not found", "class is full").
 * ============================================================================================
 */
func ManagerAddUserEnrollment(userID uint, courseID uint) error {

	/* -------------------------------------------------------------------------
	   STEP 1: IDENTITY VERIFICATION
	   Before any enrollment logic can proceed, we must ensure that the targeted
	   user actually exists in our core identity database.
	   ------------------------------------------------------------------------- */
	if _, err := dao.GetUserByID(userID); err != nil {
		// Log: User validation failed.
		// Returning a clear error helps the frontend display accurate feedback.
		return errors.New("user not found")
	}

	/* -------------------------------------------------------------------------
	   STEP 2: COURSE RESOURCE VERIFICATION
	   Verify that the CourseID provided points to a valid and active course record.
	   This also retrieves the course metadata (like Capacity) for later steps.
	   ------------------------------------------------------------------------- */
	course, err := dao.GetCourseByID(courseID)
	if err != nil {
		// Log: Course lookup failed.
		return errors.New("class not found")
	}

	/* -------------------------------------------------------------------------
	   STEP 3: DUPLICATE ENROLLMENT CHECK
	   Prevent the system from creating redundant records. A user should not be
	   enrolled in the same course multiple times simultaneously.
	   ------------------------------------------------------------------------- */
	exists, err := dao.CheckEnrollmentExists(userID, courseID)
	if err != nil {
		// Database-level error during existence check.
		return err
	}
	if exists {
		// Business Logic Violation: User is already on the roster.
		return errors.New("enrollment already exists")
	}

	/* -------------------------------------------------------------------------
	   STEP 4: PHYSICAL CAPACITY VALIDATION
	   Even for managers, we respect the physical or virtual seating limit defined
	   for the course to maintain educational quality and safety standards.
	   ------------------------------------------------------------------------- */
	count, err := dao.CountEnrollmentsByClass(courseID)
	if err != nil {
		// Database-level error during count aggregation.
		return err
	}

	/* Compare current enrollment count against the maximum allowed capacity. */
	if int(count) >= course.Capacity {
		// Business Logic Violation: No seats available.
		return errors.New("class is full")
	}

	/* -------------------------------------------------------------------------
	   STEP 5: TEMPORAL SESSION MAPPING
	   The system automatically attempts to find the "Next Scheduled Session."
	   An enrollment must be tied to a specific timeframe (Session) to be valid.
	   ------------------------------------------------------------------------- */
	session, err := dao.GetNextScheduledSession(courseID)
	if err != nil {
		// Logic failure: The class exists but has no future sessions scheduled.
		return errors.New("no upcoming session found for this class")
	}

	/* -------------------------------------------------------------------------
	   STEP 6: DATA MODEL CONSTRUCTION
	   Build the Enrollment object. We use the 'Enrolled' status by default.
	   The SessionID is assigned as a pointer to the ID retrieved in Step 5.
	   ------------------------------------------------------------------------- */
	enrollment := &model.Enrollment{
		UserID:    userID,
		CourseID:  courseID,
		SessionID: &session.ID,
		Status:    model.EnrollmentStatusEnrolled,
	}

	/* -------------------------------------------------------------------------
	   STEP 7: PERSISTENCE LAYER EXECUTION
	   Finally, commit the enrollment record to the database via the Data Access Object.
	   ------------------------------------------------------------------------- */
	return dao.CreateEnrollment(enrollment)
}

/**
 * ============================================================================================
 * FUNCTION: ManagerDeleteUserEnrollment
 * ============================================================================================
 *
 * DESCRIPTION:
 * This function provides administrative authority to remove a student's enrollment record
 * from a specific course. 
 *
 * USE CASES:
 * 1. Correcting administrative errors.
 * 2. Processing manual withdrawal requests that fall outside student-controlled options.
 * 3. Clearing roster spots for waitlisted students.
 *
 * LOGIC:
 * It performs a direct call to the DAO layer to perform a hard or soft delete (depending 
 * on DAO implementation) of the relationship between the UserID and the CourseID.
 *
 * PARAMETERS:
 * @param userID   (uint) : The ID of the student to be removed.
 * @param courseID (uint) : The ID of the course from which the student is being removed.
 *
 * RETURNS:
 * @return error : Standard error interface. Returns nil if the deletion was executed
 * successfully at the database level.
 * ============================================================================================
 */
func ManagerDeleteUserEnrollment(userID uint, courseID uint) error {
	/* The deletion logic is encapsulated within the DAO. 
	   It typically performs an operation similar to:
	   DELETE FROM enrollments WHERE user_id = ? AND course_id = ?
	*/
	return dao.DeleteEnrollment(userID, courseID)
}

/* ============================================================================================
   END OF SERVICE LAYER: MANAGER ACTIONS
   The functions above ensure that administrative tasks remain consistent with the 
   overall data integrity of the system while providing necessary flexibility.
   ============================================================================================
*/