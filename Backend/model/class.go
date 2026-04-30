/*
Package model defines the core data structures and database schema for the application.
This package utilizes GORM (Go Object Relational Mapper) tags to define table constraints,
indices, and relationships for an underlying SQLite/Postgres database.

The architecture follows a "Template-Instance" pattern:
1. Course: The abstract template (e.g., "Monday Yoga").
2. ClassSession: The concrete occurrence (e.g., "Yoga on May 5th, 2026").
3. Enrollment: The transactional link between a User and a Session.
*/
package model

import (
	"time"
)

// ============================================================================
// ENROLLMENT LIFECYCLE CONSTANTS
// ============================================================================
// These constants represent the finite state machine (FSM) of a student's
// relationship with a specific class session.

const (
	// EnrollmentStatusEnrolled indicates a successful transaction where a seat
	// is reserved, but the event has not occurred yet.
	EnrollmentStatusEnrolled = "enrolled"

	// EnrollmentStatusAttended is a terminal state set by an instructor or 
	// manager when the student is physically or virtually present.
	EnrollmentStatusAttended = "attended"

	// EnrollmentStatusMissed is a terminal state used for reporting and analytics
	// when a student fails to check-in for a reserved spot ("No-show").
	EnrollmentStatusMissed = "missed"
)

// ============================================================================
// MODEL: COURSE (The Template)
// ============================================================================

// Course represents the blueprint of an educational or fitness offering.
// In the database, this serves as the "Parent" table to ClassSessions.
//
// GORM Attributes:
// - primaryKey: Defines the unique identifier.
// - column: Explicitly maps the Go struct field to a DB column name.
// - not null: Ensures data integrity at the database level.
type Course struct {
	// ID is the unique internal identifier. Using 'uint' is standard for 
	// auto-incrementing primary keys in GORM to optimize indexing.
	ID uint `gorm:"primaryKey;autoIncrement;column:id" json:"id"`

	// CourseName is the human-readable title. 
	// Example: "Intro to Advanced Calculus" or "HIIT Cardio".
	CourseName string `gorm:"column:course_name;not null" json:"name"`

	// CourseCode is a business-logic identifier often used for searching or 
	// integration with external registration systems.
	CourseCode string `gorm:"column:course_code;not null" json:"course_code"`

	// Description allows for long-form text. In SQLite, this maps to TEXT, 
	// allowing for detailed syllabus or equipment requirements.
	Description string `gorm:"column:description" json:"description"`

	// StartTime defines the recurring start time. Note: TimeOnly is a custom 
	// type (usually handled via scanner/valuer) to store HH:MM:SS without dates.
	StartTime TimeOnly `gorm:"column:start_time;type:time" json:"start_time"`

	// EndTime defines the recurring end time. Used to calculate duration 
	// and prevent scheduling overlaps.
	EndTime TimeOnly `gorm:"column:end_time;type:time" json:"end_time"`

	// Capacity sets the upper limit for participation. This value is usually 
	// copied to new ClassSessions but can be overridden there.
	Capacity int `gorm:"column:capacity;not null" json:"capacity"`

	// Duration represents the total minutes. Storing this as an int simplifies 
	// frontend calculations for progress bars or calendar blocks.
	Duration int `gorm:"column:duration" json:"duration"`

	// Category helps in filtering the course catalog (e.g., "Language", "Tech").
	Category string `gorm:"column:category" json:"category"`

	// Weekday represents the schedule (e.g., "Monday"). This is used by 
	// background workers to automate the generation of ClassSessions.
	Weekday string `gorm:"column:weekday" json:"weekday"`

	// Instructor stores the name of the lead teacher. In advanced versions, 
	// this might be a Foreign Key to a 'Users' table with an 'Instructor' role.
	Instructor string `gorm:"column:instructor" json:"instructor"`

	// Spot is a transient/virtual field. The 'gorm:"-"' tag tells GORM to 
	// ignore this field during DB migrations and CRUD operations.
	// It is calculated dynamically: Capacity - (Number of Enrollments).
	Spot int `gorm:"-" json:"spot"`
}

// ============================================================================
// MODEL: CLASS SESSION (The Concrete Event)
// ============================================================================

// ClassSession represents a single occurrence of a Course on a specific date.
// If a Course runs "Every Monday", there will be one ClassSession record for 
// every Monday of the year.
type ClassSession struct {
	// ID is the unique ID for this specific date's event.
	ID uint `gorm:"primaryKey;autoIncrement;column:id" json:"id"`

	// CourseID creates a Many-to-One relationship. Many sessions belong to one course.
	// Indexed for high-performance joins when viewing a course's schedule.
	CourseID uint `gorm:"column:course_id;not null;index" json:"course_id"`

	// SessionDate is stored as a string (YYYY-MM-DD) to simplify date-based 
	// filtering and prevent Timezone-related offset bugs in SQLite.
	SessionDate string `gorm:"column:session_date;not null;index" json:"session_date"`

	// StartAt is a full RFC3339 timestamp. This is critical for calendar 
	// integrations (iCal/Google Calendar).
	StartAt time.Time `gorm:"column:start_at;not null" json:"start_at"`

	// EndAt is the full completion timestamp.
	EndAt time.Time `gorm:"column:end_at;not null" json:"end_at"`

	// Status tracks the availability of the session. 
	// "scheduled": Normal operations.
	// "canceled": Session won't happen (e.g., instructor illness).
	// "completed": Historical record.
	Status string `gorm:"column:status;not null;default:'scheduled'" json:"status"`

	// Capacity allows for exceptions. (e.g., "This specific room is smaller, 
	// so this Monday only, capacity is 10 instead of 20").
	Capacity int `gorm:"column:capacity" json:"capacity"`

	// Course is a GORM Relationship (Belongs To). 
	// By including this, we can use db.Preload("Course") to get parent details
	// in a single SQL query (Eager Loading).
	Course Course `gorm:"foreignKey:CourseID" json:"course"`

	// Spot is calculated at runtime to show remaining seats to students.
	Spot int `gorm:"-" json:"spot"`
}

// ============================================================================
// MODEL: ENROLLMENT (The Join Table)
// ============================================================================

// Enrollment handles the registration logic. It is a "Rich Join Table" 
// because it doesn't just link IDs; it carries its own metadata (Status, Time).
type Enrollment struct {
	// ID is the unique registration reference number.
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// UserID identifies the student. References the User model.
	UserID uint `gorm:"column:user_id;not null" json:"user_id"`

	// CourseID identifies the course template the user joined.
	CourseID uint `gorm:"column:course_id;not null" json:"course_id"`

	// SessionID links the user to the specific date they are attending.
	// This is a pointer (*uint) to allow it to be NULL if your business 
	// logic allows "Open Enrollments" not tied to a specific date.
	SessionID *uint `gorm:"column:session_id" json:"session_id"`

	// User Relationship: Allows accessing student profile from an enrollment record.
	User User `gorm:"foreignKey:UserID" json:"user"`

	// Course Relationship: Allows accessing course name/code from an enrollment record.
	Course Course `gorm:"foreignKey:CourseID" json:"course"`

	// Session Relationship: Allows accessing date/time details from an enrollment record.
	Session *ClassSession `gorm:"foreignKey:SessionID" json:"session"`

	// Status uses the constants defined at the top of this file.
	// Provides an audit trail: Did the user just sign up, or did they actually attend?
	Status string `gorm:"column:status;not null" json:"status"`

	// EnrollTime uses 'autoCreateTime' to let GORM handle the timestamping.
	// This is vital for "First-come, first-served" waitlist logic.
	EnrollTime time.Time `gorm:"column:enroll_time;autoCreateTime" json:"enroll_time"`
}

// ============================================================================
// TABLE NAME OVERRIDES
// ============================================================================
// By default, GORM pluralizes struct names (e.g., Course becomes "courses").
// These methods enforce Singular naming conventions to match strict DB schemas.

// TableName for Course ensures the table is named "Course" in the DB.
func (Course) TableName() string {
	return "Course"
}

// TableName for ClassSession ensures the table is named "ClassSession" in the DB.
func (ClassSession) TableName() string {
	return "ClassSession"
}

// TableName for Enrollment ensures the table is named "Enrollment" in the DB.
func (Enrollment) TableName() string {
	return "Enrollment"
}

// ============================================================================
// DATA TRANSFER OBJECTS (DTOs)
// ============================================================================
// DTOs are used for API request/response parsing. They decouple the internal
// database structure from the external API contract.

// EnrollmentRequest defines the expected JSON payload when a student 
// hits the POST /enroll endpoint.
type EnrollmentRequest struct {
	// CourseID is validated using the 'binding:"required"' tag. 
	// If the field is missing in the JSON body, the Gin framework 
	// will automatically return a 400 Bad Request.
	CourseID uint `json:"course_id" binding:"required"`
}

/* SUMMARY OF CONSTRAINTS AND INDEXES:
1. Every model uses an unsigned integer (uint) for ID to maximize range.
2. Indexing is applied to 'course_id' and 'session_date' because these
   are the most frequently queried columns in a scheduling application.
3. 'autoIncrement' is used to ensure the database handles ID generation, 
   preventing ID collisions in concurrent environments.
*/