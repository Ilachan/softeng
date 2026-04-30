package dao

import (
	"errors"
	"time"

	"my-course-backend/db"
	"my-course-backend/model"

	"gorm.io/gorm/clause"
)

// GetCourseByID retrieves a specific course record from the database using its unique primary key.
// Returns a pointer to the Course model and an error if the record is not found.
func GetCourseByID(id uint) (*model.Course, error) {
	var class model.Course
	// Using First() retrieves the first record ordered by primary key; returns error if empty.
	if err := db.DB.First(&class, id).Error; err != nil {
		return nil, err
	}
	return &class, nil
}

// ListClasses fetches all available course records, sorted by their start time in ascending order.
func ListClasses() ([]model.Course, error) {
	var classes []model.Course
	// Order by start_time to ensure the UI displays the timeline chronologically.
	if err := db.DB.Order("start_time ASC").Find(&classes).Error; err != nil {
		return nil, err
	}
	return classes, nil
}

// ListCategories returns a unique list of course categories.
// It filters out null values, empty strings, and whitespace-only entries.
func ListCategories() ([]string, error) {
	var categories []string
	if err := db.DB.Model(&model.Course{}).
		Distinct("category").
		// Ensure data integrity by filtering out invalid or blank category strings.
		Where("category IS NOT NULL AND TRIM(category) != ''").
		Order("category ASC").
		Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// GetUserByID fetches a user profile by their ID.
func GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if err := db.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// CheckEnrollmentExists determines if a user is already signed up for the next available session of a course.
// It uses a subquery to target only 'scheduled' sessions to prevent past enrollments from blocking new sign-ups.
func CheckEnrollmentExists(userID uint, courseID uint) (bool, error) {
	var count int64
	if err := db.DB.Model(&model.Enrollment{}).
		Where(`user_id = ? AND course_id = ? AND session_id IN (
            SELECT id FROM ClassSession WHERE course_id = ? AND status = 'scheduled' ORDER BY session_date ASC LIMIT 1
        )`, userID, courseID, courseID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetNextScheduledSession finds the earliest upcoming 'scheduled' session for a given course ID.
func GetNextScheduledSession(courseID uint) (*model.ClassSession, error) {
	var session model.ClassSession
	// Filtering by status='scheduled' and ordering by date provides the immediate next class.
	if err := db.DB.Where("course_id = ? AND status = 'scheduled'", courseID).
		Order("session_date ASC").
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// CountEnrollmentsByClass calculates total active enrollments for the next upcoming session of a course.
func CountEnrollmentsByClass(courseID uint) (int64, error) {
	var count int64
	if err := db.DB.Model(&model.Enrollment{}).
		Where(`course_id = ? AND session_id IN (
            SELECT id FROM ClassSession WHERE course_id = ? AND status = 'scheduled' ORDER BY session_date ASC LIMIT 1
        )`, courseID, courseID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CreateEnrollment inserts a new enrollment record into the database.
func CreateEnrollment(enrollment *model.Enrollment) error {
	return db.DB.Create(enrollment).Error
}

// DeleteEnrollment removes an enrollment for a user. 
// It strictly targets the next 'scheduled' session where the status is currently 'enrolled'.
func DeleteEnrollment(userID uint, courseID uint) error {
	result := db.DB.Where(`user_id = ? AND course_id = ? AND session_id IN (
        SELECT id FROM ClassSession WHERE course_id = ? AND status = 'scheduled' ORDER BY session_date ASC LIMIT 1
    ) AND status = 'enrolled'`, userID, courseID, courseID).
		Delete(&model.Enrollment{})

	if result.Error != nil {
		return result.Error
	}
	// If no rows were affected, it means the user wasn't enrolled or the session isn't in 'scheduled' status.
	if result.RowsAffected == 0 {
		return errors.New("enrollment not found")
	}
	return nil
}

// ListEnrollmentsByClass retrieves all students signed up for a course, including nested User profile data.
func ListEnrollmentsByClass(courseID uint) ([]model.Enrollment, error) {
	var enrollments []model.Enrollment
	if err := db.DB.Where("course_id = ?", courseID).
		Preload("User"). // Eager load the User relationship.
		Find(&enrollments).Error; err != nil {
		return nil, err
	}
	return enrollments, nil
}

// ListEnrolledCoursesByUser returns all course objects associated with a specific user's active enrollments.
func ListEnrolledCoursesByUser(userID uint) ([]model.Course, error) {
	var courses []model.Course
	// Perform an INNER JOIN to filter courses that exist in the Enrollment table for this user.
	if err := db.DB.Joins("INNER JOIN Enrollment ON Enrollment.course_id = Course.id").
		Where("Enrollment.user_id = ? AND Enrollment.status = ?", userID, model.EnrollmentStatusEnrolled).
		Order("Course.start_time ASC").
		Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

// CreateCourse adds a new course definition to the system.
func CreateCourse(course *model.Course) error {
	return db.DB.Create(course).Error
}

// UpdateCourse performs a full update of an existing course's fields.
func UpdateCourse(course *model.Course) error {
	return db.DB.Save(course).Error
}

// DeleteCourseByID removes a course from the database via its primary key.
func DeleteCourseByID(id uint) error {
	return db.DB.Delete(&model.Course{}, id).Error
}

// CreateDailyActivity logs a user's daily activity. 
// Uses OnConflict clause to prevent duplicate entries for the same activity event.
func CreateDailyActivity(activity *model.UserDailyActivity) error {
	return db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(activity).Error
}

// BackfillUserDailyActivityFromEnrollments synchronizes the analytics table (UserDailyActivity) 
// with the actual attendance recorded in the Enrollment table.
// It ensures that only past, 'attended' sessions are calculated into the user's history.
func BackfillUserDailyActivityFromEnrollments(userID uint) error {
	// Raw SQL for efficient bulk processing.
	// COALESCE handles cases where the session date might be missing by falling back to enroll_time.
	query := `
        INSERT INTO UserDailyActivity (enrollment_id, user_id, course_id, activity_date, created_at)
        SELECT e.id, e.user_id, e.course_id,
            COALESCE(cs.session_date, DATE(e.enroll_time)),
            CURRENT_TIMESTAMP
        FROM Enrollment e
        LEFT JOIN ClassSession cs ON cs.id = e.session_id
        WHERE e.user_id = ?
        AND e.status = 'attended'
        AND COALESCE(cs.session_date, DATE(e.enroll_time)) <= DATE('now')
        AND NOT EXISTS (
            SELECT 1
            FROM UserDailyActivity uda
            WHERE uda.enrollment_id = e.id
        )
    `
	return db.DB.Exec(query, userID).Error
}

// GetUserActivityStats calculates the total number of classes attended and unique days active 
// within a specific time window.
func GetUserActivityStats(userID uint, fromDate time.Time, toDate time.Time) (int64, int64, error) {
	type statsResult struct {
		TotalClasses int64
		ActiveDays   int64
	}

	var result statsResult
	err := db.DB.Model(&model.UserDailyActivity{}).
		// Count total records and unique activity dates for frequency metrics.
		Select("COUNT(*) as total_classes, COUNT(DISTINCT activity_date) as active_days").
		Where("user_id = ? AND activity_date BETWEEN ? AND ?", userID, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02")).
		Scan(&result).Error
	if err != nil {
		return 0, 0, err
	}

	return result.TotalClasses, result.ActiveDays, nil
}

// GetUserTotalTime sums up the total duration (in minutes) of all courses a user attended during the date range.
func GetUserTotalTime(userID uint, fromDate time.Time, toDate time.Time) (int64, error) {
	type totalTimeResult struct {
		TotalTime int64
	}

	var result totalTimeResult
	// Joins UserDailyActivity with Course to access the 'duration' field of each course.
	err := db.DB.Table("UserDailyActivity AS uda").
		Select("COALESCE(SUM(COALESCE(c.duration, 0)), 0) AS total_time").
		Joins("INNER JOIN Course c ON c.id = uda.course_id").
		Where("uda.user_id = ? AND uda.activity_date BETWEEN ? AND ?", userID, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02")).
		Scan(&result).Error
	if err != nil {
		return 0, err
	}

	return result.TotalTime, nil
}

// GetUserDailyActivitySummary generates a breakdown of how many classes were taken on each individual date.
func GetUserDailyActivitySummary(userID uint, fromDate time.Time, toDate time.Time) ([]model.DailyActivitySummary, error) {
	var daily []model.DailyActivitySummary

	err := db.DB.Table("UserDailyActivity AS uda").
		Select(`DATE(uda.activity_date) AS date,
            COUNT(*) AS classes`).
		Where("uda.user_id = ? AND uda.activity_date BETWEEN ? AND ?", userID, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02")).
		Group("DATE(uda.activity_date)").
		Order("DATE(uda.activity_date) ASC").
		Scan(&daily).Error
	if err != nil {
		return nil, err
	}

	return daily, nil
}

// GetUserCategoryActivitySummary aggregates attendance data by course category.
// If a category is empty, it labels the group as 'Uncategorized'.
func GetUserCategoryActivitySummary(userID uint, fromDate time.Time, toDate time.Time) ([]model.CategoryActivitySummary, error) {
	var categories []model.CategoryActivitySummary

	err := db.DB.Table("UserDailyActivity AS uda").
		// Clean and label empty categories on the fly for cleaner reporting.
		Select(`COALESCE(NULLIF(TRIM(c.category), ''), 'Uncategorized') AS category,
            COUNT(*) AS classes`).
		Joins("INNER JOIN Course c ON c.id = uda.course_id").
		Where("uda.user_id = ? AND uda.activity_date BETWEEN ? AND ?", userID, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02")).
		Group("COALESCE(NULLIF(TRIM(c.category), ''), 'Uncategorized')").
		Order("classes DESC, category ASC").
		Scan(&categories).Error
	if err != nil {
		return nil, err
	}

	return categories, nil
}

// GetEnrollment fetches a single enrollment record based on user and course IDs.
func GetEnrollment(userID uint, courseID uint) (*model.Enrollment, error) {
	var enrollment model.Enrollment
	if err := db.DB.Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&enrollment).Error; err != nil {
		return nil, err
	}
	return &enrollment, nil
}

// UpdateEnrollmentStatus modifies the status (e.g., 'enrolled' -> 'attended') for a user's enrollment.
// Returns true if the record was successfully updated.
func UpdateEnrollmentStatus(userID uint, courseID uint, status string) (bool, error) {
	tx := db.DB.Model(&model.Enrollment{}).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		Update("status", status)

	if tx.Error != nil {
		return false, tx.Error
	}
	// RowsAffected allows the caller to know if the ID pair actually existed.
	return tx.RowsAffected > 0, nil
}