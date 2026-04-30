package model

import "time"

// UserDailyActivity tracks individual attendance or participation records.
// It serves as the granular data source for generating user performance metrics.
type UserDailyActivity struct {
	// ID: Internal tracking number for the activity record (Primary Key).
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// EnrollmentID: Links to the specific enrollment record.
	// Index Constraint: Included in 'idx_sda_enrollment_date' to prevent 
	// duplicate activity entries for the same enrollment on the same date.
	EnrollmentID uint `gorm:"column:enrollment_id;not null;index:idx_sda_enrollment_date,unique" json:"enrollment_id"`
	
	// UserID: The identifier of the user who performed the activity.
	// Indexed for fast lookup when generating personal user reports.
	UserID       uint `gorm:"column:user_id;not null;index" json:"user_id"`
	
	// CourseID: The identifier of the course associated with this activity.
	// Allows for course-specific engagement analytics.
	CourseID     uint `gorm:"column:course_id;not null;index" json:"course_id"`

	// ActivityDate: The calendar date (YYYY-MM-DD) when the activity occurred.
	// Unique Index: Combined with EnrollmentID to ensure data integrity.
	ActivityDate time.Time `gorm:"column:activity_date;type:date;not null;index:idx_sda_enrollment_date,unique;index" json:"activity_date"`
	
	// CreatedAt: Audit timestamp recording when this record was first persisted.
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides GORM's default naming convention to keep it singular.
func (UserDailyActivity) TableName() string {
	return "UserDailyActivity"
}

// DailyActivitySummary represents a simplified data point for time-series charts.
// This is typically used to render "Classes over Time" bar or line graphs.
type DailyActivitySummary struct {
	// Date: The formatted date string (e.g., "2023-10-27").
	Date    string `json:"date"`
	
	// Classes: The count of class sessions completed on this specific date.
	Classes int64  `json:"classes"`
}

// CategoryActivitySummary provides a breakdown of user interest by course types.
// Ideal for rendering Pie Charts or Radar Charts on the dashboard.
type CategoryActivitySummary struct {
	// Category: The name of the course category (e.g., "Yoga", "HIIT", "Strength").
	Category   string  `json:"category"`
	
	// Classes: Total number of classes attended within this specific category.
	Classes    int64   `json:"classes"`
	
	// Percentage: The proportional share of this category relative to all activities.
	Percentage float64 `json:"percentage"`
}

// UserAnalyticsResponse is the master Data Transfer Object (DTO) for the Analytics API.
// It bundles high-level stats and detailed chart data into a single payload.
type UserAnalyticsResponse struct {
	// UserID: Confirms which user these statistics belong to.
	UserID       uint                      `json:"user_id"`
	
	// Range: The human-readable timeframe (e.g., "last_7_days", "current_month").
	Range        string                    `json:"range"`
	
	// FromDate / ToDate: The explicit ISO-8601 date boundaries for the data set.
	FromDate     string                    `json:"from_date"`
	ToDate       string                    `json:"to_date"`
	
	// TotalClasses: Cumulative count of all sessions attended within the range.
	TotalClasses int64                     `json:"total_classes"`
	
	// TotalTime: Cumulative duration (usually in minutes) spent in classes.
	TotalTime    int64                     `json:"total_time"`
	
	// ActiveDays: Number of unique days the user had at least one activity.
	ActiveDays   int64                     `json:"active_days"`
	
	// Daily: A slice of data points for chronological activity visualization.
	Daily        []DailyActivitySummary    `json:"daily"`
	
	// Categories: A slice of data points for categorical distribution visualization.
	Categories   []CategoryActivitySummary `json:"categories"`
}