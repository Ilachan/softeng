package dao

import (
	"time"

	"my-course-backend/db"
	"my-course-backend/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetManagerInviteCodeForUpdate retrieves a specific invite code record while applying a row-level lock.
//
// Concurrency & Locking:
// It uses the "FOR UPDATE" clause (via GORM's clause.Locking) to prevent race conditions.
// This ensures that if two processes try to use the same invite code simultaneously, 
// the database will force them to serialize, preventing "double-use" bugs.
//
// Parameters:
// - tx: An active GORM transaction instance (required for row-level locking).
// - code: The unique alphanumeric string representing the invite code.
//
// Returns:
// - A pointer to the ManagerInviteCode model or an error if the code is invalid or not found.
func GetManagerInviteCodeForUpdate(tx *gorm.DB, code string) (*model.ManagerInviteCode, error) {
	var invite model.ManagerInviteCode
	// Apply 'SELECT ... FOR UPDATE' to lock the row until the transaction commits or rolls back.
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("code = ?", code).
		First(&invite).Error; err != nil {
		return nil, err
	}
	return &invite, nil
}

// MarkInviteCodeUsed updates the status of an invite code to prevent future reuse.
//
// Data Integrity:
// This function records the exact timestamp of use and the email of the person who claimed it.
// It should typically be called within the same transaction as GetManagerInviteCodeForUpdate.
func MarkInviteCodeUsed(tx *gorm.DB, id uint, inviteeEmail string) error {
	now := time.Now()
	return tx.Model(&model.ManagerInviteCode{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        "used",
			"used_at":       now,
			"invitee_email": inviteeEmail,
		}).Error
}

// WithTx is a wrapper utility that encapsulates a function within a database transaction.
// 
// Transaction Management:
// It handles the lifecycle of a transaction (Begin, Commit, and Rollback). 
// If the provided function 'fn' returns an error, the transaction is automatically rolled back.
// Otherwise, it is committed to the database.
func WithTx(fn func(tx *gorm.DB) error) error {
	return db.DB.Transaction(fn)
}

// ListUsersPaged retrieves a slice of users using pagination to optimize network and memory usage.
//
// Pagination Logic:
// 1. Count: It first calculates the 'total' number of records to help the frontend build pagination UI.
// 2. Preload: It eagerly loads the "Role" association to avoid N+1 query overhead.
// 3. Ordering: Sorts by 'id ASC' to ensure a stable and consistent order across different pages.
//
// Parameters:
// - limit: The maximum number of user records to return per request.
// - offset: The number of records to skip (usually: (pageNumber - 1) * limit).
//
// Returns:
// - A slice of User models, the total count of users, and any potential error.
func ListUsersPaged(limit int, offset int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	// Get the total count for the frontend to calculate total pages.
	if err := db.DB.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch the subset of data for the current page.
	if err := db.DB.
		Preload("Role"). // Join/Fetch Role data for each user in this batch.
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// ListEnrollmentsByUser retrieves all course enrollments associated with a specific user ID.
//
// Data Relationship:
// It uses Preload("Course") to ensure that for every enrollment record returned,
// the detailed course information (Title, Description, etc.) is also populated.
func ListEnrollmentsByUser(userID uint) ([]model.Enrollment, error) {
	var enrollments []model.Enrollment
	if err := db.DB.
		Where("user_id = ?", userID).
		Preload("Course"). // Eager load the Course entity associated with the Enrollment.
		Find(&enrollments).Error; err != nil {
		return nil, err
	}
	return enrollments, nil
}