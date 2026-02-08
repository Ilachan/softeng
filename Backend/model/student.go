package model

import "time"

// Role represents the role table in the database.
type Role struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleName string `gorm:"unique;not null" json:"role_name"`
}

// Student represents the student user in the database.
type Student struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Email     string    `gorm:"unique;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"` // json:"-" ensures the password is never returned in the API response
	AvatarURL string    `json:"avatar_url"`
	RoleID    uint      `json:"role_id"` // Foreign Key
	Role      Role      `gorm:"foreignKey:RoleID" json:"role"` // Association (Belongs To)
	CreatedAt time.Time `json:"created_at"`
}

// TableName overrides the default table name to "Student".
func (Student) TableName() string {
	return "Student"
}

// TableName overrides the default table name to "Role".
func (Role) TableName() string {
	return "Role"
}

// RegisterInput captures the parameters sent by the frontend for registration.
type RegisterInput struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"` // Password must be at least 6 characters
}

// LoginInput captures the parameters sent by the frontend for login.
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}