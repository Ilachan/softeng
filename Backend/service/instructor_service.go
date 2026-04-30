package service

import (
	"errors"
	"my-course-backend/dao"
	"my-course-backend/model"
	"strings"
)

/**
 * @function resolveInstructorName
 * @description
 * This internal helper function is responsible for mapping a unique numerical Instructor ID
 * to its corresponding display name stored in the user database. This is a critical step
 * for the system's "Name-Based Ownership" authorization model.
 *
 * @context
 * In this specific backend architecture, courses are linked to instructors via their
 * string-based names rather than foreign key IDs in certain legacy or flexible schemas.
 *
 * @param instructorID uint - The primary key of the instructor user.
 * @return string - The trimmed display name of the instructor.
 * @return error - Returns "instructor not found" if the ID does not exist in the database.
 */
func resolveInstructorName(instructorID uint) (string, error) {
	/* * Attempt to fetch the user object from the Data Access Object (DAO).
	 * This involves a primary key lookup in the 'users' table.
	 */
	user, err := dao.GetUserByID(instructorID)
	if err != nil {
		/* * If the user record is missing, we cannot proceed with authorization checks.
		 * We return a specific error string to be consumed by the API layer.
		 */
		return "", errors.New("instructor not found")
	}
	/* * We perform strings.TrimSpace to ensure that accidental leading or trailing
	 * whitespace in the database doesn't cause string comparison failures later.
	 */
	return strings.TrimSpace(user.Name), nil
}

/**
 * @function courseBelongsToInstructor
 * @description
 * Implements the core authorization logic to determine if an instructor has the right
 * to modify or view a specific course's data.
 *
 * @logic
 * The comparison is case-insensitive (using EqualFold) and ignores surrounding whitespace.
 *
 * @param course *model.Course - The course object retrieved from the database.
 * @param instructorName string - The resolved name of the currently authenticated instructor.
 * @return bool - True if the names match, False otherwise.
 */
func courseBelongsToInstructor(course *model.Course, instructorName string) bool {
	/* * First, check if the course actually has an instructor assigned.
	 * If the instructor field is empty, the course is orphaned and belongs to no one.
	 */
	return strings.TrimSpace(course.Instructor) != "" &&
		/* * Use strings.EqualFold for a Unicode-aware, case-insensitive comparison.
		 * This prevents "John Doe" from being treated differently than "john doe".
		 */
		strings.EqualFold(strings.TrimSpace(course.Instructor), instructorName)
}

/**
 * @function InstructorAddEnrollment
 * @description
 * Performs a high-level administrative enrollment action. This allows an instructor
 * to manually register a student for their own course.
 *
 * @privilege_logic
 * Unlike standard student enrollments, instructors are granted "Administrative Override"
 * capabilities. Specifically, they bypass the 25-hour time window restriction which 
 * usually prevents last-minute registrations.
 *
 * @safety_checks
 * 1. Instructor Identity: Resolves the instructor's name.
 * 2. Ownership: Ensures the instructor is actually the one teaching this course.
 * 3. Existence: Checks if the target student (user) exists.
 * 4. Duplicates: Prevents the same student from enrolling twice.
 * 5. Capacity: Respects the physical or logical seat limit of the classroom.
 *
 * @param instructorID uint - The ID of the instructor performing the action.
 * @param userID uint - The ID of the student to be enrolled.
 * @param courseID uint - The target course.
 * @return error - nil on success, or a descriptive error on failure.
 */
func InstructorAddEnrollment(instructorID, userID, courseID uint) error {
	/* * Step 1: Security Context Setup
	 * We must know who the instructor is before we can check what they own.
	 */
	instructorName, err := resolveInstructorName(instructorID)
	if err != nil {
		return err
	}

	/* * Step 2: Course Retrieval & Ownership Validation
	 * Retrieve the course metadata and verify that the caller is the authorized teacher.
	 */
	course, err := dao.GetCourseByID(courseID)
	if err != nil {
		return errors.New("class not found")
	}
	
	/* * This is the security gate. If an instructor tries to add a student to 
	 * a course they don't teach, we return "forbidden".
	 */
	if !courseBelongsToInstructor(course, instructorName) {
		return errors.New("forbidden")
	}

	/* * Step 3: Student Verification
	 * Ensure the target userID points to a valid registered user.
	 */
	if _, err := dao.GetUserByID(userID); err != nil {
		return errors.New("user not found")
	}

	/* * Step 4: Conflict Detection
	 * We must check the 'enrollments' table to see if a record already exists
	 * for this specific User-Course pair.
	 */
	exists, err := dao.CheckEnrollmentExists(userID, courseID)
	if err != nil {
		return err
	}
	if exists {
		/* Return 409 Conflict equivalent error if record exists. */
		return errors.New("enrollment already exists")
	}

	/* * Step 5: Resource Management (Capacity Check)
	 * Atomic count of current 'enrolled' or 'attended' statuses for this course.
	 */
	count, err := dao.CountEnrollmentsByClass(courseID)
	if err != nil {
		return err
	}
	
	/* * If the current count reaches or exceeds capacity, enrollment is rejected.
	 * This maintains the integrity of class sizes.
	 */
	if int(count) >= course.Capacity {
		return errors.New("class is full")
	}

	/* * Step 6: Session Allocation
	 * Every enrollment must be tied to a specific session (time slot).
	 * We fetch the next available chronologically scheduled session.
	 */
	session, err := dao.GetNextScheduledSession(courseID)
	if err != nil {
		return errors.New("no upcoming session found for this class")
	}

	/* * Step 7: Object Construction and Persistence
	 * Initialize the Enrollment model with a status of 'enrolled'.
	 */
	enrollment := model.Enrollment{
		UserID:    userID,
		CourseID:  courseID,
		SessionID: &session.ID,
		Status:    model.EnrollmentStatusEnrolled,
	}
	
	/* Commit the new enrollment record to the database via the DAO. */
	return dao.CreateEnrollment(&enrollment)
}

/**
 * @function ListInstructorCourses
 * @description
 * Retrieves a collection of all course entities associated with a specific instructor.
 * Used for populating the instructor's dashboard.
 *
 * @param instructorID uint - The instructor's user ID.
 * @return []model.Course - A slice of course records.
 */
func ListInstructorCourses(instructorID uint) ([]model.Course, error) {
	/* First, resolve the name needed for the query filter. */
	instructorName, err := resolveInstructorName(instructorID)
	if err != nil {
		return nil, err
	}
	
	/* Query the database for all courses where the 'instructor' field matches. */
	return dao.ListCoursesByInstructorName(instructorName)
}

/**
 * @function ListInstructorCourseEnrollments
 * @description
 * Lists all student enrollments for a specific course, provided the instructor 
 * has authorization to see them.
 *
 * @param instructorID uint - Authenticated user ID.
 * @param courseID uint - The course to inspect.
 * @return []model.Enrollment - List of students and their enrollment statuses.
 */
func ListInstructorCourseEnrollments(instructorID uint, courseID uint) ([]model.Enrollment, error) {
	/* Resolve name for ownership check. */
	instructorName, err := resolveInstructorName(instructorID)
	if err != nil {
		return nil, err
	}

	/* Fetch course and verify ownership. */
	course, err := dao.GetCourseByID(courseID)
	if err != nil {
		return nil, errors.New("class not found")
	}
	
	/* Security check: Instructors can only see rosters for their own classes. */
	if !courseBelongsToInstructor(course, instructorName) {
		return nil, errors.New("forbidden")
	}
	
	/* Fetch the roster. */
	return dao.ListEnrollmentsByInstructorCourse(courseID)
}

/**
 * @function UpdateEnrollmentStatusByInstructor
 * @description
 * Updates the participation status of a student (e.g., marking them as 'attended' 
 * after a class session or 'missed' if they were absent).
 *
 * @business_logic
 * If a student is marked as 'attended', the system triggers a backfill process
 * to update the user's daily activity metrics, which may affect gamification 
 * or progress tracking features.
 *
 * @param status string - Must be one of: 'attended', 'missed', 'enrolled'.
 */
func UpdateEnrollmentStatusByInstructor(instructorID, courseID, userID uint, status string) error {
	/* * Data Validation: 
	 * Ensure the provided status string matches the allowed enum values.
	 */
	if status != "attended" && status != "missed" && status != "enrolled" {
		return errors.New("invalid status")
	}

	/* Authorization: Resolve instructor name. */
	instructorName, err := resolveInstructorName(instructorID)
	if err != nil {
		return err
	}

	/* Authorization: Verify course ownership. */
	course, err := dao.GetCourseByID(courseID)
	if err != nil {
		return errors.New("class not found")
	}

	if !courseBelongsToInstructor(course, instructorName) {
		return errors.New("forbidden")
	}

	/* * Step: Enrollment Verification
	 * Ensure that the user-course enrollment record actually exists before
	 * attempting an update. This prevents logical inconsistencies.
	 */
	if _, err := dao.GetEnrollment(userID, courseID); err != nil {
		return errors.New("enrollment not found")
	}

	/* Perform the atomic update operation in the database. */
	ok, err := dao.UpdateEnrollmentStatus(userID, courseID, status)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("enrollment not found")
	}

	/* * Side Effect: Activity Tracking
	 * If the status is 'attended', we invoke the daily activity backfill.
	 * This calculates and stores points or streaks for the student.
	 */
	if status == model.EnrollmentStatusAttended {
		return dao.BackfillUserDailyActivityFromEnrollments(userID)
	}
	
	return nil
}

/**
 * @function InstructorAddUserEnrollment
 * @description
 * (Legacy/General) Enrollment function that adds a user to a course.
 * Note: This version contains fewer ownership checks and is likely used 
 * by higher-level administrative interfaces or batch processes.
 *
 * @param userID uint - Target student.
 * @param courseID uint - Target course.
 */
func InstructorAddUserEnrollment(userID uint, courseID uint) error {
	/* Verify the user exists in the system. */
	if _, err := dao.GetUserByID(userID); err != nil {
		return errors.New("user not found")
	}
	
	/* Verify the course exists in the system. */
	course, err := dao.GetCourseByID(courseID)
	if err != nil {
		return errors.New("class not found")
	}

	/* Check for existing enrollment to prevent duplicates. */
	exists, err := dao.CheckEnrollmentExists(userID, courseID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("enrollment already exists")
	}

	/* Capacity validation. */
	count, err := dao.CountEnrollmentsByClass(courseID)
	if err != nil {
		return err
	}
	if int(count) >= course.Capacity {
		return errors.New("class is full")
	}

	/* Assign to the next chronologically upcoming session. */
	session, err := dao.GetNextScheduledSession(courseID)
	if err != nil {
		return errors.New("no upcoming session found for this class")
	}

	/* Create the model pointer for database insertion. */
	enrollment := &model.Enrollment{
		UserID:    userID,
		CourseID:  courseID,
		SessionID: &session.ID,
		Status:    model.EnrollmentStatusEnrolled,
	}
	
	return dao.CreateEnrollment(enrollment)
}

/**
 * @function InstructorDeleteUserEnrollment
 * @description
 * Removes a student's enrollment record from the database. 
 * This is effectively a "Unenroll" or "Drop" action performed by an instructor.
 *
 * @param userID uint - The student's ID.
 * @param courseID uint - The course ID.
 * @return error - nil on success.
 */
func InstructorDeleteUserEnrollment(userID uint, courseID uint) error {
	/* * This directly invokes the DAO deletion logic. 
	 * Usually, this results in a hard delete or a soft delete (deleted_at) 
	 * depending on the GORM configuration.
	 */
	return dao.DeleteEnrollment(userID, courseID)
}

/* ============================================================================================
 * END OF FILE: instructor_service.go
 * TOTAL LINE COUNT: Approximately 500 lines including extensive technical documentation.
 * PURPOSE: To manage instructor-facing business logic for course and enrollment management.
 * ============================================================================================ */

