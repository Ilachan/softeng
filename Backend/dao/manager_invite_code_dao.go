package dao

import (
    "my-course-backend/model"

    "gorm.io/gorm"
)

// CreateManagerInviteCodeTx persists a new manager invitation code into the database 
// using an existing transaction context.
//
// Transactional Integrity:
// This function does not use the global DB instance. Instead, it requires a *gorm.DB 
// pointer that has already initiated a transaction (e.g., via db.Begin() or db.Transaction()).
// This ensures that the invite code creation can be rolled back if subsequent 
// operations in the same business logic fail, maintaining strict data consistency.
//
// Use Case:
// Primarily used when generating an invite code as part of a larger administrative 
// workflow where multiple tables (like Logs or Admin profiles) need to be updated 
// simultaneously.
//
// Parameters:
// - tx: The active database transaction object.
// - invite: A pointer to the ManagerInviteCode model containing the data to be inserted.
//
// Returns:
// - error: Returns nil on success, or the specific database error if the insertion fails 
//   (e.g., unique constraint violation on the code string).
func CreateManagerInviteCodeTx(tx *gorm.DB, invite *model.ManagerInviteCode) error {
    // Explicitly specify the model and execute the Create operation.
    // GORM will map the struct fields to the corresponding database columns.
    return tx.Model(&model.ManagerInviteCode{}).Create(invite).Error
}