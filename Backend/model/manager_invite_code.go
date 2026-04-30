package model

import "time"

// ManagerInviteCode represents the schema for the "Manager_Invite_Code" table in SQLite.
// It tracks the lifecycle of an invitation code from generation to redemption.
type ManagerInviteCode struct {
    // ID: Unique identifier for each invite record (Primary Key)
    ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
    
    // Code: The unique alphanumeric string used for registration authentication
    Code         string     `gorm:"column:code" json:"code"`
    
    // InviterID: The ID of the SuperManager who generated this code (Nullable)
    InviterID    *uint      `gorm:"column:inviter_id" json:"inviter_id"`
    
    // InviteeEmail: Optional target email. If set, only this email can use the code
    InviteeEmail *string    `gorm:"column:invitee_email" json:"invitee_email"`
    
    // Status: Current state of the code (e.g., "unused", "used", "expired")
    Status       *string    `gorm:"column:status" json:"status"`
    
    // CreatedAt: Timestamp when the invitation record was first created
    CreatedAt    *time.Time `gorm:"column:created_at" json:"created_at"`
    
    // ExpiredAt: The deadline after which the code is no longer valid (Mandatory)
    ExpiredAt    *time.Time `gorm:"column:expired_at;not null" json:"expired_at"`
    
    // UsedAt: Timestamp recording when the invitee successfully registered
    UsedAt       *time.Time `gorm:"column:used_at" json:"used_at"`
}

// TableName overrides the default GORM table naming strategy to match the existing schema.
func (ManagerInviteCode) TableName() string {
    return "Manager_Invite_Code"
}

// ManagerRegisterInput defines the required fields for a new manager to sign up.
// It includes validation tags for email format and password strength.
type ManagerRegisterInput struct {
    // Name: Full name or display name of the manager
    Name       string `json:"name" binding:"required"`
    
    // Email: Official email address, must be a valid email format
    Email      string `json:"email" binding:"required,email"`
    
    // Password: Minimum 6 characters required for security
    Password   string `json:"password" binding:"required,min=6"`
    
    // InviteCode: The specific token provided by a SuperManager
    InviteCode string `json:"invite_code" binding:"required"`
}

// CreateManagerInviteInput is used by SuperManagers to generate new invitation tokens.
type CreateManagerInviteInput struct {
    // InviteeEmail: Optional. If provided, restricts the code to a specific recipient.
    // Use 'omitempty' to allow general-purpose invite codes.
    InviteeEmail string `json:"invitee_email" binding:"omitempty,email"`

    // ExpireHours: Sets the validity duration of the code in hours.
    // Constraints: Must be at least 1 hour and no more than 720 hours (30 days).
    ExpireHours int `json:"expire_hours" binding:"required,min=1,max=720"` 
}