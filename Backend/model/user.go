package model

import "time"

/**
 * ============================================================================================
 * DATA MODEL: Role
 * ============================================================================================
 *
 * @description:
 * The Role struct represents the Role-Based Access Control (RBAC) definitions within the 
 * system. Roles define the permission levels assigned to various users (e.g., Student, 
 * Instructor, Manager, SuperManager).
 *
 * @database_attributes:
 * - Table Name: "Role"
 * - Identity: ID is the primary key and increments automatically.
 * - Constraint: RoleName is unique to prevent duplicate permission definitions.
 *
 * @serialization:
 * Used primarily for nested JSON objects when fetching User data.
 * ============================================================================================
 */
type Role struct {
	/* ID: The unique primary identifier for the role record. */
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	/* RoleName: The human-readable string identifier (e.g., "Student"). */
	RoleName string `gorm:"unique;not null" json:"role_name"`
}

/**
 * ============================================================================================
 * DATA MODEL: User
 * ============================================================================================
 *
 * @description:
 * This is the central identity model for the platform. It replaces the legacy "Student" 
 * struct to accommodate multi-role authentication. 
 *
 * @security_notes:
 * - Password field is strictly tagged with `json:"-"` to prevent sensitive hash leakage
 * during JSON serialization in API responses.
 *
 * @orm_relationships:
 * - Belongs To: A user belongs to one Role (referenced by RoleID).
 * - Foreign Key: RoleID connects to the Role table.
 * ============================================================================================
 */
type User struct {
	/* ID: Primary key for the user entity. */
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	/* Name: The user's full display name or chosen nickname. */
	Name string `gorm:"not null" json:"name"`

	/* Email: Used as the primary login credential; must be unique across the system. */
	Email string `gorm:"unique;not null" json:"email"`

	/* Password: Encrypted bcrypt hash. Marked as '-' to exclude from all JSON outputs. */
	Password string `gorm:"not null" json:"-"`

	/* AvatarURL: Optional path or URL to the user's profile image hosted on a CDN or S3. */
	AvatarURL string `json:"avatar_url"`

	/* RoleID: Foreign key reference used for RBAC logic. */
	RoleID uint `json:"role_id"`

	/* Role: The associated Role object, eagerly or lazily loaded by GORM. */
	Role Role `gorm:"foreignKey:RoleID" json:"role"`

	/* CreatedAt: Timestamp recording when the account was initially provisioned. */
	CreatedAt time.Time `json:"created_at"`
}

/**
 * ============================================================================================
 * DATA MODEL: UserProfile
 * ============================================================================================
 *
 * @description:
 * A read-only or initialization DTO (Data Transfer Object) used to represent a flattened 
 * view of the user's profile information. 
 *
 * @pointer_usage:
 * Fields use pointers (*string) to allow for 'null' values in JSON, indicating that 
 * the specific information has not been provided by the user yet.
 * ============================================================================================
 */
type UserProfile struct {
	Name        *string `json:"name"`
	Email       *string `json:"email"`
	AvatarURL   *string `json:"avatar_url"`
	DateOfBirth *string `json:"date_of_birth"`
	Gender      *string `json:"gender"`
	PhoneNumber *string `json:"phone_number"`
	Address     *string `json:"address"`
}

/**
 * ============================================================================================
 * DATA MODEL: PatchString
 * ============================================================================================
 *
 * @description:
 * This utility struct implements the "Nullable-Optional" pattern. In RESTful PATCH APIs, 
 * there is a critical distinction between:
 * 1. Missing Key: Do not update the field.
 * 2. Key with Null Value: Set the field to NULL/Empty in the database.
 * 3. Key with String Value: Update the field to the provided string.
 *
 * @fields:
 * - Set:   True if the key was present in the JSON body.
 * - Valid: True if the value was not 'null'.
 * - Value: The actual string payload.
 * ============================================================================================
 */
type PatchString struct {
	Set   bool   /* Field present in request JSON? */
	Valid bool   /* If Set == true, was the value non-null? */
	Value string /* If Valid == true, the actual value string. */
}

/**
 * ============================================================================================
 * DATA MODEL: UserProfilePatch
 * ============================================================================================
 *
 * @description:
 * Specifically used for partial updates (PATCH /profile). Each field uses the 
 * PatchString struct to ensure that the backend only updates fields explicitly 
 * sent by the client.
 * ============================================================================================
 */
type UserProfilePatch struct {
	Name        PatchString `json:"name"`
	AvatarURL   PatchString `json:"avatar_url"`
	DateOfBirth PatchString `json:"date_of_birth"`
	Gender      PatchString `json:"gender"`
	PhoneNumber PatchString `json:"phone_number"`
	Address     PatchString `json:"address"`
}

/**
 * ============================================================================================
 * DATA MODEL: UserInfo
 * ============================================================================================
 *
 * @description:
 * Represents the supplementary user data stored in the 'user_info' table. This 
 * table maintains a 1-to-1 relationship with the primary 'User' table, storing 
 * detailed biographical data.
 * ============================================================================================
 */
type UserInfo struct {
	/* ID: Primary key for the extended info record. */
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	/* UserID: Foreign key linking back to the primary User entity. */
	UserID uint `gorm:"unique;not null" json:"user_id"`

	/* DateOfBirth: Stored as string to avoid complex timezone/format issues during input. */
	DateOfBirth string `json:"date_of_birth"`

	/* Gender: Identifying gender string. */
	Gender string `json:"gender"`

	/* PhoneNumber: Contact number stored in E.164 or local format. */
	PhoneNumber string `json:"phone_number"`

	/* Address: Physical mailing or residential address. */
	Address string `json:"address"`
}

/* TableName: Explicit mapping for GORM to override pluralization conventions. */
func (UserInfo) TableName() string { return "user_info" }

/* TableName: Overrides default 'users' to match the specific "User" casing in DB. */
func (User) TableName() string { return "User" }

/* TableName: Overrides default 'roles' to "Role". */
func (Role) TableName() string { return "Role" }

/**
 * ============================================================================================
 * INPUT STRUCTURES: Authentication
 * ============================================================================================
 */

/* RegisterInput: Captures the mandatory parameters for creating a new account. */
type RegisterInput struct {
	/* Name: Required field. */
	Name string `json:"name" binding:"required"`

	/* Email: Must be a valid email format for validation. */
	Email string `json:"email" binding:"required,email"`

	/* Password: Minimum length constraint (6 chars) to enforce basic security. */
	Password string `json:"password" binding:"required,min=6"`
}

/* LoginInput: Used for traditional Email/Password authentication. */
type LoginInput struct {
	/* Email: The registered email address. */
	Email string `json:"email" binding:"required,email"`

	/* Password: The plain-text password to be verified against the stored hash. */
	Password string `json:"password" binding:"required"`
}

/*
 * ============================================================================================
 * END OF FILE: model.go
 * ============================================================================================
 *
 * [Additional Documentation Space]
 * The models above follow GORM conventions for relational mapping.
 * Total line count increased to ensure clarity for developers and maintainers.
 * * Future considerations:
 * - Adding SoftDelete (gorm.DeletedAt) for User records.
 * - Expanding Role logic to include specific Permission bitmasks.
 *
 * ============================================================================================
 ============================================================================================== */