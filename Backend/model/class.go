package model

import "time"

// Enrollment Status Constants
// These constants define the lifecycle of a student's participation in a course or session.
const (
    // EnrollmentStatusEnrolled: The student has successfully signed up but has not yet attended.
    EnrollmentStatusEnrolled = "enrolled"
    // EnrollmentStatusAttended: The student's presence has been verified for the class.
    EnrollmentStatusAttended = "attended"
    // EnrollmentStatusMissed: The student failed to show up for the scheduled class.
    EnrollmentStatusMissed   = "missed"
)

// Course represents the master template for a course in the SQLite database.
// It defines the recurring properties and metadata for educational offerings.
type Course struct {
    // ID: Unique identifier for the course (Primary Key with Auto-Increment).
    ID uint `gorm:"primaryKey;autoIncrement;column:id" json:"id"`

    // CourseName: The descriptive title of the course (e.g., "Advanced Yoga").
    CourseName string `gorm:"column:course_name;not null" json:"name"`
    
    // CourseCode: A unique shorthand identifier (e.g., "CS101", "YOGA-01").
    CourseCode string `gorm:"column:course_code;not null" json:"course_code"`

    // Description: Detailed information about the course content and requirements.
    Description string `gorm:"column:description" json:"description"`

    // StartTime: The standard daily/weekly time when the class begins.
    StartTime TimeOnly `gorm:"column:start_time;type:time" json:"start_time"`
    
    // EndTime: The standard daily/weekly time when the class concludes.
    EndTime   TimeOnly `gorm:"column:end_time;type:time" json:"end_time"`

    // Capacity: Maximum number of students allowed to enroll in this course.
    Capacity int `gorm:"column:capacity;not null" json:"capacity"`

    // Duration: Total length of the class session in minutes.
    Duration int    `gorm:"column:duration" json:"duration"`
    
    // Category: The grouping label for the course (e.g., "Fitness", "Arts").
    Category string `gorm:"column:category" json:"category"`
    
    // Weekday: The specific day of the week the course occurs (e.g., "Monday").
    Weekday  string `gorm:"column:weekday" json:"weekday"`

    // Instructor: The name or ID of the primary teacher for this course.
    Instructor string `gorm:"column:instructor" json:"instructor"`

    // Spot: A virtual field (not in DB) used to calculate remaining availability in real-time.
    Spot int `gorm:"-" json:"spot"`
}

// ClassSession represents a specific, timed instance of a recurring Course.
// While a Course is the template, a ClassSession is the actual event on a specific date.
type ClassSession struct {
    // ID: Unique identifier for the specific session instance.
    ID          uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
    
    // CourseID: Foreign Key linking this session back to its parent Course template.
    CourseID    uint      `gorm:"column:course_id;not null;index" json:"course_id"`
    
    // SessionDate: The specific calendar date for this session in YYYY-MM-DD format.
    SessionDate string    `gorm:"column:session_date;not null;index" json:"session_date"`
    
    // StartAt: The absolute start timestamp (date + time).
    StartAt     time.Time `gorm:"column:start_at;not null" json:"start_at"`
    
    // EndAt: The absolute end timestamp (date + time).
    EndAt       time.Time `gorm:"column:end_at;not null" json:"end_at"`
    
    // Status: Lifecycle of the session (e.g., "scheduled", "canceled", "completed").
    Status      string    `gorm:"column:status;not null;default:'scheduled'" json:"status"`
    
    // Capacity: Overrides Course.Capacity if set; allows per-session size adjustments.
    Capacity    int       `gorm:"column:capacity" json:"capacity"`

    // Course: Relation object to access parent Course details via Eager Loading.
    Course Course `gorm:"foreignKey:CourseID" json:"course"`
    
    // Spot: Calculated field for available seats remaining for this specific session.
    Spot   int    `gorm:"-" json:"spot"`
}

// Enrollment acts as the many-to-many join table connecting Users to Courses and Sessions.
// It tracks which user is registered for which specific class event.
type Enrollment struct {
    // ID: Unique identifier for the enrollment record.
    ID        uint  `gorm:"primaryKey;autoIncrement" json:"id"`
    
    // UserID: Foreign Key referencing the User who is enrolling.
    UserID    uint  `gorm:"column:user_id;not null" json:"user_id"`
    
    // CourseID: Foreign Key referencing the Course template.
    CourseID  uint  `gorm:"column:course_id;not null" json:"course_id"`
    
    // SessionID: Pointer to ClassSession; nullable to support legacy general-course signups.
    SessionID *uint `gorm:"column:session_id" json:"session_id"`

    // User: Relation object for the person enrolled.
    User    User          `gorm:"foreignKey:UserID" json:"user"`
    
    // Course: Relation object for the course metadata.
    Course  Course        `gorm:"foreignKey:CourseID" json:"course"`
    
    // Session: Relation object for the specific class instance (nullable).
    Session *ClassSession `gorm:"foreignKey:SessionID" json:"session"`

    // Status: Current state of this specific enrollment (Enrolled/Attended/Missed).
    Status     string    `gorm:"column:status;not null" json:"status"`
    
    // EnrollTime: Automatically populated timestamp when the record is created.
    EnrollTime time.Time `gorm:"column:enroll_time;autoCreateTime" json:"enroll_time"`
}

// TableName overrides: Explicitly naming tables to avoid GORM's default pluralization logic.

func (Course) TableName() string {
    return "Course"
}

func (ClassSession) TableName() string {
    return "ClassSession"
}

func (Enrollment) TableName() string {
    return "Enrollment"
}

// EnrollmentRequest is the Data Transfer Object (DTO) for processing incoming signup requests.
type EnrollmentRequest struct {
    // CourseID: The ID of the course the user wishes to join. Required by the validator.
    CourseID uint `json:"course_id" binding:"required"`
}