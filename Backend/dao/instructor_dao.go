package dao

import (
    "my-course-backend/db"
    "my-course-backend/model"
)

// ListCoursesByInstructorName retrieves all course records from the database 
// that are associated with a specific instructor's name.
// 
// Implementation Details:
// 1. Data Normalization: It applies LOWER() and TRIM() on both the database column 
//    and the input parameter to ensure a case-insensitive and whitespace-resilient match.
// 2. Sorting: The resulting list is ordered by 'start_time' in ascending order 
//    to provide a chronological timeline of courses.
//
// Parameters:
// - name: The string representing the instructor's name to search for.
//
// Returns:
// - A slice of model.Course objects matching the criteria.
// - An error object if the database query fails.
func ListCoursesByInstructorName(name string) ([]model.Course, error) {
    var courses []model.Course
    
    // Execute the query using GORM's fluent API.
    // We use raw SQL fragments within Where() for complex string manipulation.
    if err := db.DB.
        Where("LOWER(TRIM(instructor)) = LOWER(TRIM(?))", name).
        Order("start_time ASC").
        Find(&courses).Error; err != nil {
        return nil, err
    }
    
    return courses, nil
}

// ListEnrollmentsByInstructorCourse fetches all student enrollment records 
// for a specific course identified by its unique ID.
//
// Relationship Handling:
// This function utilizes GORM's 'Preload' feature to eagerly load the associated 
// 'User' model data. This prevents the "N+1 query problem" by fetching user 
// details (like names or emails) in a single batch or join, making it efficient 
// for displaying student lists.
//
// Parameters:
// - courseID: The primary key (uint) of the course.
//
// Returns:
// - A slice of model.Enrollment objects, each containing an embedded User object.
// - An error if the database operation encounters an issue.
func ListEnrollmentsByInstructorCourse(courseID uint) ([]model.Enrollment, error) {
    var enrollments []model.Enrollment
    
    // Filter enrollments by the specific course ID.
    // Preload("User") ensures the Enrollment.User struct is populated.
    if err := db.DB.Where("course_id = ?", courseID).
        Preload("User").
        Find(&enrollments).Error; err != nil {
        return nil, err
    }
    
    return enrollments, nil
}