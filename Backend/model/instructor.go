package model

// Instructor represents an instructor's detailed profile within the system.
// There is a strict one-to-one relationship between an Instructor and a User 
// where the User must have a role_id associated with instructor permissions (typically role_id=4).
type Instructor struct {
    // ID is the unique internal identifier for the instructor record (Primary Key).
    // It is configured to auto-increment in the database.
    ID     uint   `gorm:"primaryKey;autoIncrement;column:id" json:"id"`

    // UserID serves as a Foreign Key linking to the User model.
    // The 'uniqueIndex' ensures the one-to-one constraint at the database level.
    // 'not null' ensures every instructor profile must be attached to a valid user account.
    UserID uint   `gorm:"column:user_id;not null;uniqueIndex" json:"user_id"`

    // Name stores the full name or display name of the instructor.
    // This can be used to store a professional name different from the account username.
    Name   string `gorm:"column:name" json:"name"`

    // Bio provides a short biography or professional description of the instructor.
    // This is typically displayed on public profile pages or course descriptions.
    Bio    string `gorm:"column:bio" json:"bio"`

    // User is the embedded User model instance populated via GORM's Preload feature.
    // It defines the relationship structure: Instructor belongs to a User.
    User User `gorm:"foreignKey:UserID" json:"user"`
}

// TableName explicitly defines the database table name as "Instructor".
// This prevents GORM from using the default pluralized name (instructors).
func (Instructor) TableName() string { 
    return "Instructor" 
}