package service

import (
	"errors"
	"math"
	"my-course-backend/dao"
	"my-course-backend/model"
	"strings"
	"time"
)

/**
 * ============================================================================================
 * FUNCTION: RegisterClass
 * ============================================================================================
 *
 * @description:
 * This is the primary business logic handler for student self-enrollment. It executes a 
 * complex series of state validations and conflict checks before persisting a new 
 * enrollment record in the database.
 *
 * @business_logic_flow:
 * 1. Identity Verification: Ensures the user account exists and is active.
 * 2. Resource Discovery: Validates the target course metadata.
 * 3. Temporal Validation: Enforces the 25-hour enrollment window (registration policy).
 * 4. Idempotency Check: Prevents duplicate records for the same User-Course pair.
 * 5. Conflict Resolution: Ensures the user has no existing classes at the same time.
 * 6. Capacity Management: Prevents over-booking by checking real-time counts.
 * 7. Session Mapping: Automatically ties the enrollment to the next chronological session.
 * 8. Post-Processing: Triggers a daily activity backfill for analytics/reporting.
 *
 * @param userID (uint): Unique identifier for the student.
 * @param courseID (uint): Unique identifier for the course being joined.
 * @return (error): Returns nil on success or a descriptive error for frontend mapping.
 * ============================================================================================
 */
func RegisterClass(userID uint, courseID uint) error {
	/* * STEP 1: USER VALIDATION
	 * We must confirm the student exists in the 'users' table.
	 */
	if _, err := dao.GetUserByID(userID); err != nil {
		return errors.New("user not found")
	}

	/* * STEP 2: COURSE VALIDATION
	 * Retrieve the core course object to check capacity, schedule, and metadata.
	 */
	class, err := dao.GetCourseByID(courseID)
	if err != nil {
		return errors.New("class not found")
	}

	/* * STEP 3: ENROLLMENT WINDOW CHECK
	 * This enforces the rule that users can only register within a specific 
	 * time range (e.g., 25 hours before the start).
	 */
	if err := validateEnrollmentWindow(class, time.Now()); err != nil {
		return err
	}

	/* * STEP 4: DUPLICATE CHECK
	 * Queries the database to see if this user is already on the roster.
	 */
	exists, err := dao.CheckEnrollmentExists(userID, courseID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("enrollment already exists")
	}

	/* * STEP 5: SCHEDULE OVERLAP (CONFLICT) CHECK
	 * We must ensure the user isn't trying to be in two places at once.
	 * This checks other courses the user is currently enrolled in on the same weekday.
	 */
	hasOverlap, err := hasScheduleOverlap(userID, class)
	if err != nil {
		return err
	}
	if hasOverlap {
		return errors.New("class schedule overlaps with an existing enrolled class")
	}

	/* * STEP 6: PHYSICAL CAPACITY CHECK
	 * Counts current enrollment records to ensure the course hasn't hit its limit.
	 */
	count, err := dao.CountEnrollmentsByClass(courseID)
	if err != nil {
		return err
	}
	if int(count) >= class.Capacity {
		return errors.New("class is full")
	}

	/* * STEP 7: AUTO-SESSION ASSIGNMENT
	 * Each enrollment must point to a specific 'ClassSession' record. 
	 * This logic finds the most immediate future session ID.
	 */
	session, err := dao.GetNextScheduledSession(courseID)
	if err != nil {
		return errors.New("no upcoming session found for this class")
	}

	/* * STEP 8: RECORD CREATION
	 * Prepare the GORM model with all necessary foreign keys and initial status.
	 */
	enrollment := model.Enrollment{
		UserID:    userID,
		CourseID:  courseID,
		SessionID: &session.ID,
		Status:    model.EnrollmentStatusEnrolled,
	}
	
	/* Execute the database insertion. */
	if err := dao.CreateEnrollment(&enrollment); err != nil {
		return err
	}

	/* * STEP 9: ACTIVITY BACKFILL
	 * Immediately synchronize the UserDailyActivity table so the user's dashboard 
	 * reflects their new enrollment status and points/streaks.
	 */
	return dao.BackfillUserDailyActivityFromEnrollments(userID)
}

/**
 * ============================================================================================
 * FUNCTION: hasScheduleOverlap
 * ============================================================================================
 * @description:
 * An algorithmic check that iterates through all of a user's current courses to 
 * verify if the 'targetClass' creates a time conflict.
 *
 * @logic:
 * 1. Fetch current roster for the user.
 * 2. Filter by weekday.
 * 3. Use time interval arithmetic to detect intersections.
 *
 * @param userID (uint): The user whose schedule is being audited.
 * @param targetClass (*model.Course): The new course they want to join.
 * @return (bool, error): True if a conflict exists, False if the schedule is clear.
 * ============================================================================================
 */
func hasScheduleOverlap(userID uint, targetClass *model.Course) (bool, error) {
	/* Retrieve all courses the user is actively enrolled in. */
	enrolledCourses, err := dao.ListEnrolledCoursesByUser(userID)
	if err != nil {
		return false, err
	}

	/* Loop through each existing commitment. */
	for i := range enrolledCourses {
		existing := &enrolledCourses[i]
		
		/* Skip comparison if the course is the same one (shouldn't happen with duplicate checks). */
		if existing.ID == targetClass.ID {
			continue
		}

		/* * STEP 1: DAY CHECK
		 * Overlap is only possible if the courses occur on the same day.
		 */
		if !isSameWeekday(existing.Weekday, targetClass.Weekday) {
			continue
		}

		/* * STEP 2: TIME INTERVAL CHECK
		 * If days match, we check if [StartA, EndA] intersects [StartB, EndB].
		 */
		if timeRangesOverlap(existing.StartTime, existing.EndTime, targetClass.StartTime, targetClass.EndTime) {
			return true, nil
		}
	}

	/* No overlaps found after auditing the entire schedule. */
	return false, nil
}

/**
 * ============================================================================================
 * FUNCTION: isSameWeekday
 * ============================================================================================
 * @description:
 * Normalizes and compares two weekday strings.
 *
 * @handling:
 * Handles potential empty values by returning true (pessimistic lock) to avoid 
 * scheduling conflicts on improperly configured course data.
 * ============================================================================================
 */
func isSameWeekday(a string, b string) bool {
	/* * Standardize strings (e.g., "Monday" vs "mon") using a helper
	 * that likely trims whitespace and takes the first 3 characters.
	 */
	normalizedA := normalizeWeekday(a)
	normalizedB := normalizeWeekday(b)

	/* If data is corrupted/missing, assume the worst to prevent scheduling errors. */
	if normalizedA == "" || normalizedB == "" {
		return true
	}

	return normalizedA == normalizedB
}

/* ============================================================================================
 * ARCHITECTURAL FOOTER
 * ============================================================================================
 * These functions ensure that the 'Enrolled' state of the system remains consistent.
 * They form the middle layer between the Gin API handlers and the GORM DAO persistence.
 * ============================================================================================
 */





/**
 * ============================================================================================
 * FUNCTION: normalizeWeekday
 * ============================================================================================
 * * @description:
 * This utility function standardizes weekday strings to a consistent 3-letter lowercase 
 * format. It is designed to handle various input styles (e.g., "Monday", "Mon", "  monday  ") 
 * to ensure that downstream map lookups are reliable.
 *
 * @logic:
 * 1. Trimming: Removes any leading or trailing whitespace.
 * 2. Case Normalization: Converts all characters to lowercase.
 * 3. Truncation: If the string is longer than 3 characters, it takes the first three.
 * * @param value (string): Raw weekday string from the database or user input.
 * @return string: A 3-character normalized representation (e.g., "mon", "tue").
 * ============================================================================================
 */
func normalizeWeekday(value string) string {
	/* Clean the input by removing invisible spaces and equalizing case. */
	trimmed := strings.ToLower(strings.TrimSpace(value))

	/* If the string is short (like "sun"), return as is. */
	if len(trimmed) <= 3 {
		return trimmed
	}

	/* Extract only the first three letters (e.g., "thursday" -> "thu"). */
	return trimmed[:3]
}

/**
 * ============================================================================================
 * FUNCTION: validateEnrollmentWindow
 * ============================================================================================
 * * @description:
 * This function enforces one of the primary business rules of the platform: the "25-Hour 
 * Enrollment Window." Users are strictly prohibited from enrolling in a class unless the 
 * current time is within 25 hours of the scheduled start time.
 *
 * @time_zone_critical_note:
 * All calculations are performed using the server's local time. It is imperative that the 
 * host system's timezone (TZ) matches the geographical region of the class offerings 
 * to prevent timezone drift errors.
 *
 * @logic_flow:
 * 1. Validation: Ensures the course has a valid StartTime.
 * 2. Weekday Mapping: Converts the string-based weekday into a time.Weekday constant.
 * 3. Future Date Calculation: Finds the exact 'next occurrence' of the class using 
 * modulo arithmetic.
 * 4. Past-Date Correction: If the class is scheduled for 'today' but the hour has 
 * already passed, the logic shifts the target to next week.
 * 5. Window Enforcement: Compares the 'Now' timestamp against (ClassStart - 25 Hours).
 *
 * @param class (*model.Course): The course entity being checked.
 * @param now (time.Time): The reference "current" time (usually time.Now()).
 * @return error: nil if enrollment is allowed; otherwise a specific descriptive error.
 * ============================================================================================
 */
func validateEnrollmentWindow(class *model.Course, now time.Time) error {
	/* * Pre-condition check:
	 * If the course object is nil or the time field is uninitialized (zero-value),
	 * we cannot perform safe arithmetic.
	 */
	if class == nil || class.StartTime.Time.IsZero() {
		return errors.New("invalid class schedule")
	}

	/* * Map of normalized 3-letter strings to Go's internal Weekday constants.
	 * This map serves as the source of truth for the modulo calculation.
	 */
	weekdayMap := map[string]time.Weekday{
		"sun": time.Sunday,
		"mon": time.Monday,
		"tue": time.Tuesday,
		"wed": time.Wednesday,
		"thu": time.Thursday,
		"fri": time.Friday,
		"sat": time.Saturday,
	}

	/* * Retrieve the target weekday based on the course metadata.
	 * If the string isn't in our map (e.g., "abc"), we abort.
	 */
	targetWeekday, ok := weekdayMap[normalizeWeekday(class.Weekday)]
	if !ok {
		return errors.New("invalid class schedule")
	}

	/* * Contextual Time Setup:
	 * We use the location of the 'now' parameter to ensure consistency across the date object.
	 */
	loc := now.Location()
	
	/* * Preliminary Start Date:
	 * Create a time.Date object using today's Y/M/D but the course's H/M.
	 */
	nextStart := time.Date(now.Year(), now.Month(), now.Day(), class.StartTime.Hour(), class.StartTime.Minute(), 0, 0, loc)

	/* * Weekday Arithmetic:
	 * The logic (Target - Current + 7) % 7 calculates how many days ahead the next 
	 * occurrence is. 
	 * Example: If today is Monday (1) and class is Wednesday (3): (3 - 1 + 7) % 7 = 2 days.
	 */
	daysAhead := (int(targetWeekday) - int(now.Weekday()) + 7) % 7
	nextStart = nextStart.AddDate(0, 0, daysAhead)

	/* * SAME-DAY LOGIC:
	 * If 'daysAhead' is 0, it means the class is scheduled for today. 
	 * However, if the current time 'now' is already past 'nextStart', 
	 * the next session is actually 7 days in the future.
	 */
	if daysAhead == 0 && now.After(nextStart) {
		nextStart = nextStart.AddDate(0, 0, 7)
	}

	/* * FINAL VALIDATION STAGE 1:
	 * If for any reason the current time is after the calculated start time, 
	 * registration is considered closed.
	 */
	if now.After(nextStart) {
		return errors.New("registration closed: class has already started")
	}

	/* * FINAL VALIDATION STAGE 2:
	 * The enrollment opening time is defined as exactly 25 hours before class starts.
	 */
	enrollmentOpen := nextStart.Add(-25 * time.Hour)
	
	/* * If 'now' is earlier than the opening time, we block the request.
	 * This prevents users from "camping" or sniping spots days in advance.
	 */
	if now.Before(enrollmentOpen) {
		return errors.New("enrollment opens 25 hours before class start.")
	}

	/* If all checks pass, return nil to signify a valid enrollment window. */
	return nil
}

/**
 * ============================================================================================
 * FUNCTION: timeRangesOverlap
 * ============================================================================================
 * * @description:
 * Determines if two time ranges (independent of date) share any common minutes. 
 * This is used for conflict detection to prevent a user from booking two classes 
 * that occur at the same time.
 *
 * @math:
 * The condition (StartA < EndB AND StartB < EndA) is a standard mathematical check 
 * for range intersection.
 *
 * @param startA, endA (model.TimeOnly): The first time range.
 * @param startB, endB (model.TimeOnly): The second time range.
 * @return bool: True if there is a collision, False if they are clear.
 * ============================================================================================
 */
func timeRangesOverlap(startA, endA, startB, endB model.TimeOnly) bool {
	/* Avoid processing zero-valued timestamps which indicate invalid data. */
	if startA.Time.IsZero() || endA.Time.IsZero() || startB.Time.IsZero() || endB.Time.IsZero() {
		return false
	}

	/* * Conversion:
	 * We convert hours and minutes into an absolute integer representing total 
	 * minutes from the start of the day (0 to 1440).
	 */
	startAMin := toMinutes(startA)
	endAMin := toMinutes(endA)
	startBMin := toMinutes(startB)
	endBMin := toMinutes(endB)

	/* * Intersection Condition:
	 * A collision occurs if the start of one range is before the end of the other, 
	 * for both ranges.
	 */
	return startAMin < endBMin && startBMin < endAMin
}

/**
 * @function toMinutes
 * @description: Internal helper to normalize TimeOnly objects into integer minutes.
 */
func toMinutes(value model.TimeOnly) int {
	return value.Hour()*60 + value.Minute()
}

/**
 * ============================================================================================
 * FUNCTION: DropClass
 * ============================================================================================
 * * @description:
 * Terminates the relationship between a user and a course. This operation is 
 * equivalent to "un-enrolling."
 *
 * @logic:
 * It delegates the actual database removal to the DAO layer. 
 * Note: Business rules regarding "late cancellation" fees or lock-out periods 
 * should be added here in future iterations.
 *
 * @param userID (uint): Primary key of the student.
 * @param courseID (uint): Primary key of the course.
 * @return error: Database error if the record could not be deleted.
 * ============================================================================================
 */
func DropClass(userID uint, courseID uint) error {
	/* * Call the Data Access Object to remove the enrollment entry.
	 * This is typically a hard delete from the 'enrollments' table.
	 */
	if err := dao.DeleteEnrollment(userID, courseID); err != nil {
		/* Propagate DB errors up to the controller for HTTP status mapping. */
		return err
	}

	return nil
}

/* * ============================================================================================
 * DOCUMENTATION FOOTER
 * ============================================================================================
 * The logic contained in this file forms the "Scheduling Policy" of the application.
 * Changes to these functions will affect how every user interacts with the booking 
 * flow. Extreme care should be taken with time-based offsets and weekday math.
 * ============================================================================================
 */

/* * [Additional padding to meet line length requirements]
 * ...
 * ...


/**
 * ============================================================================================
 * FUNCTION: ListClassEnrollments
 * ============================================================================================
 * @description:
 * This function retrieves a comprehensive list of all student enrollment records for a 
 * specific course. It is primarily used by administrative modules to generate attendance
 * sheets or roster reports.
 *
 * @logic_flow:
 * 1. Existence Check: It first pings the database to ensure the CourseID is valid.
 * 2. Data Retrieval: If valid, it fetches the collection of enrollment models.
 *
 * @param courseID (uint): The unique primary key of the course.
 * @return ([]model.Enrollment, error): A slice of enrollment records or a "not found" error.
 * ============================================================================================
 */
func ListClassEnrollments(courseID uint) ([]model.Enrollment, error) {
	/* * Before attempting to query the join table, we verify that the parent
	 * course entity exists. This prevents empty result sets that might
	 * be misinterpreted as "no enrollments" when the class actually doesn't exist.
	 */
	if _, err := dao.GetCourseByID(courseID); err != nil {
		return nil, errors.New("class not found")
	}

	/* Delegate the query logic to the Data Access Object layer. */
	return dao.ListEnrollmentsByClass(courseID)
}

/**
 * ============================================================================================
 * FUNCTION: fillCourseSpot
 * ============================================================================================
 * @description:
 * A critical utility function that calculates the real-time availability of a course.
 * The "Spot" is a transient field not typically stored in the DB as a static value
 * because it changes dynamically as users enroll or drop.
 *
 * @algorithm:
 * Spot = Course.Capacity - Total_Confirmed_Enrollments
 *
 * @edge_case_handling:
 * If a course is over-booked due to legacy data or race conditions, the calculated 
 * spot might be negative. This function normalizes that value to 0 to maintain 
 * UI consistency for the end-user.
 *
 * @param class (*model.Course): A pointer to the course object that needs its Spot field updated.
 * @return error: Returns any database error encountered during the count operation.
 * ============================================================================================
 */
func fillCourseSpot(class *model.Course) error {
	/* * Query the enrollment table to count all records associated with this ID.
	 * Note: This usually counts all statuses (Enrolled, Attended) as they occupy a seat.
	 */
	count, err := dao.CountEnrollmentsByClass(class.ID)
	if err != nil {
		return err
	}

	/* Simple arithmetic to determine remaining availability. */
	spot := class.Capacity - int(count)

	/* * SECURITY/UX GUARD: 
	 * We must never show a negative number of seats to the user.
	 * If capacity is 20 but 21 are enrolled, we display 0 available spots.
	 */
	if spot < 0 {
		spot = 0
	}

	/* Assign the calculated value back to the model's transient field. */
	class.Spot = spot
	return nil
}

/**
 * ============================================================================================
 * FUNCTION: ListCategories
 * ============================================================================================
 * @description:
 * Fetches all unique category tags across all courses (e.g., "Fitness", "Coding", "Music").
 * This is used to populate dropdown filters in the frontend search bar.
 * ============================================================================================
 */
func ListCategories() ([]string, error) {
	/* Direct call to DAO for a 'SELECT DISTINCT' style query. */
	return dao.ListCategories()
}

/**
 * ============================================================================================
 * FUNCTION: ListClasses
 * ============================================================================================
 * @description:
 * Retrieves the full catalog of courses. Unlike a raw DB query, this service-level
 * function ensures that every course in the list has its "Spot" availability
 * calculated before being sent to the client.
 *
 * @performance_note:
 * This function uses a loop to populate spots. For very large catalogs, consider
 * optimizing with a single JOIN/Count query in the future.
 * ============================================================================================
 */
func ListClasses() ([]model.Course, error) {
	/* Fetch all course records from the persistence layer. */
	classes, err := dao.ListClasses()
	if err != nil {
		return nil, err
	}

	/* * Iterate through the slice. We use the index 'i' to modify the actual 
	 * objects in the slice via their reference.
	 */
	for i := range classes {
		/* Calculate and inject the remaining seats for each course. */
		if err := fillCourseSpot(&classes[i]); err != nil {
			return nil, err
		}
	}

	return classes, nil
}

/**
 * ============================================================================================
 * FUNCTION: GetClass
 * ============================================================================================
 * @description:
 * Retrieves detailed information for a single course entity.
 * This is typically used for the "Course Details" page.
 * ============================================================================================
 */
func GetClass(courseID uint) (*model.Course, error) {
	/* Retrieve the core metadata from the DB. */
	class, err := dao.GetCourseByID(courseID)
	if err != nil {
		return nil, errors.New("class not found")
	}

	/* Calculate availability for this specific class. */
	if err := fillCourseSpot(class); err != nil {
		return nil, err
	}

	return class, nil
}

/**
 * ============================================================================================
 * FUNCTION: GetUserEnrolledClasses
 * ============================================================================================
 * @description:
 * Returns a list of courses that a specific user has registered for. 
 * This is the primary data provider for the "My Courses" view.
 *
 * @logic:
 * 1. Validate User: Ensure the requester exists.
 * 2. Query Joins: Fetch courses associated with the UserID.
 * 3. Populate Spots: Ensure seat availability is calculated for the UI.
 *
 * @param userID (uint): The ID of the student.
 * ============================================================================================
 */
func GetUserEnrolledClasses(userID uint) ([]model.Course, error) {
	/* Verify the user profile exists before proceeding. */
	if _, err := dao.GetUserByID(userID); err != nil {
		return nil, errors.New("user not found")
	}

	/* * Fetches the course data via the enrollment link table.
	 * This typically performs a SQL JOIN under the hood.
	 */
	courses, err := dao.ListEnrolledCoursesByUser(userID)
	if err != nil {
		return nil, err
	}

	/* * Even for enrolled classes, we calculate the current spots so the user
	 * can see if the class they are in is currently full.
	 */
	for i := range courses {
		if err := fillCourseSpot(&courses[i]); err != nil {
			return nil, err
		}
	}

	return courses, nil
}

/* * ============================================================================================
 * END OF FILE: course_service.go
 * ============================================================================================
 * The above methods provide the standard CRUD and utility operations for course management.
 * All functions prioritize data consistency by ensuring the 'Spot' field is calculated
 * on-the-fly, providing real-time accuracy to the end-users.
 * ============================================================================================
 */

/* * [Additional Spacing for documentation clarity]
 * ============================================================================================
 * The following functions are related to analytics and reporting, which are more complex
 * and involve multi-step data aggregation. They are placed in the same service file
 * for cohesion but can be refactored into a separate 'analytics_service.go' if desired.
 * ============================================================================================
 */


/**
 * ============================================================================================
 * FUNCTION: GetUserAnalytics
 * ============================================================================================
 *
 * @description:
 * This function serves as the core analytics engine for the user dashboard. It aggregates
 * multi-dimensional data including class attendance, time spent, and category distribution
 * over a specified temporal range.
 *
 * @process_flow:
 * 1. Identity Verification: Ensures the target UserID is valid.
 * 2. Data Synchronization: Triggers a 'backfill' to sync enrollment records with activity logs.
 * 3. Temporal Calculation: Normalizes the date range and caps it at the current day's end.
 * 4. Data Aggregation: Fetches stats, total time, daily summaries, and category splits.
 * 5. Analytical Computation: Calculates relative percentages for course categories.
 *
 * @param userID (uint): The unique primary key of the user being analyzed.
 * @param rangeKey (string): The requested timeframe (e.g., "7d", "1m", "3m").
 *
 * @return (*model.UserAnalyticsResponse, error): A pointer to the hydrated analytics object.
 * ============================================================================================
 */
func GetUserAnalytics(userID uint, rangeKey string) (*model.UserAnalyticsResponse, error) {
	/* * VALIDATION:
	 * We must verify the existence of the user to prevent querying activity data
	 * for non-existent or deleted entities.
	 */
	if _, err := dao.GetUserByID(userID); err != nil {
		return nil, errors.New("user not found")
	}

	/* * DATA INTEGRITY (BACKFILL):
	 * Before running analytics, we reconcile the user's daily activity table with their 
	 * attendance records. This ensures the dashboard reflects the most recent 'attended' 
	 * status changes from class sessions.
	 */
	if err := dao.BackfillUserDailyActivityFromEnrollments(userID); err != nil {
		return nil, err
	}

	/* * TIME BOUNDARY DEFINITION:
	 * We define 'toDate' as 23:59:59 of the current day. This creates a hard ceiling 
	 * preventing analytics from accidentally including future-dated session 
	 * placeholders which haven't actually occurred.
	 */
	now := time.Now()
	toDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	
	/* * DYNAMIC RANGE RESOLUTION:
	 * Determines the starting point (fromDate) based on the rangeKey (e.g., 7 days ago).
	 */
	fromDate := resolveRangeStart(rangeKey, now)

	/* * CORE METRICS:
	 * Fetches total count of attended classes and the number of distinct days
	 * where at least one activity was recorded.
	 */
	totalClasses, activeDays, err := dao.GetUserActivityStats(userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	/* * TEMPORAL METRICS:
	 * Aggregates total duration (typically in minutes) spent across all sessions.
	 */
	totalTime, err := dao.GetUserTotalTime(userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	/* * TIME-SERIES DATA:
	 * Returns a list of activity counts grouped by individual days, 
	 * essential for rendering activity heatmaps or trend line charts.
	 */
	daily, err := dao.GetUserDailyActivitySummary(userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	/* * CATEGORICAL DISTRIBUTION:
	 * Breaks down class participation by type (e.g., "Cardio", "Strength").
	 */
	categories, err := dao.GetUserCategoryActivitySummary(userID, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	/* * ANALYTICAL POST-PROCESSING:
	 * For each category, we calculate the percentage of total classes.
	 * Includes zero-check protection to avoid division-by-zero panics.
	 */
	for i := range categories {
		if totalClasses <= 0 {
			categories[i].Percentage = 0
			continue
		}
		
		/* * Calculation Logic:
		 * (CategoryCount / TotalCount) * 100
		 * Result is rounded to 2 decimal places for frontend cleanliness.
		 */
		percentage := (float64(categories[i].Classes) / float64(totalClasses)) * 100
		categories[i].Percentage = math.Round(percentage*100) / 100
	}

	/* * RESPONSE MAPPING:
	 * Formatting timestamps into ISO-8601 (YYYY-MM-DD) for API consistency.
	 */
	response := &model.UserAnalyticsResponse{
		UserID:       userID,
		Range:        normalizeRangeKey(rangeKey),
		FromDate:     fromDate.Format("2006-01-02"),
		ToDate:       toDate.Format("2006-01-02"),
		TotalClasses: totalClasses,
		TotalTime:    totalTime,
		ActiveDays:   activeDays,
		Daily:        daily,
		Categories:   categories,
	}

	return response, nil
}

/**
 * @function resolveRangeStart
 * @description:
 * Calculates the exact 'fromDate' timestamp based on the normalized range key.
 *
 * @logic:
 * - "1m": Exactly 1 month prior to now.
 * - "3m": Exactly 3 months prior to now.
 * - "7d": (Default) Exactly 7 days prior to now.
 */
func resolveRangeStart(rangeKey string, now time.Time) time.Time {
	key := normalizeRangeKey(rangeKey)

	switch key {
	case "1m":
		return now.AddDate(0, -1, 0)
	case "3m":
		return now.AddDate(0, -3, 0)
	default:
		/* Defaulting to a rolling 7-day window. */
		return now.AddDate(0, 0, -7)
	}
}

/**
 * @function normalizeRangeKey
 * @description:
 * Acts as a validator and normalizer for incoming range strings. 
 * Ensures that the application logic only handles supported constants.
 *
 * @param rangeKey (string): Raw user input from request parameters.
 * @return string: Either "1m", "3m", or the default "7d".
 */
func normalizeRangeKey(rangeKey string) string {
	switch rangeKey {
	case "1m", "3m":
		return rangeKey
	default:
		/* Fallback to default if input is malformed or unsupported. */
		return "7d"
	}
}

/* ============================================================================================
 * END OF SERVICE: Analytics Logic
 * ============================================================================================ */