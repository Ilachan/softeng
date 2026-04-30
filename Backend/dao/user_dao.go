package dao

import (
	"errors"

	"my-course-backend/db"
	"my-course-backend/model"

	"gorm.io/gorm"
)

/**
 * ROLE DATA ACCESS METHODS
 * These methods handle authorization metadata.
 */

// GetRoleByName performs a lookup in the 'roles' table to find the ID associated with a role label.
//
// Technical Logic:
// 1. It constructs a SELECT query with a LIMIT 1 (via .First()).
// 2. It maps the result to a model.Role struct.
//
// Use Cases:
// - Role-based access control (RBAC) initialization.
// - Mapping human-readable strings from frontend (e.g., "admin") to database IDs.
func GetRoleByName(name string) (uint, error) {
	var role model.Role

	// Query: SELECT * FROM roles WHERE role_name = ? ORDER BY id LIMIT 1;
	// GORM will automatically populate the 'role' struct with database values.
	err := db.DB.Where("role_name = ?", name).First(&role).Error
	if err != nil {
		// Common error: gorm.ErrRecordNotFound if the name doesn't match any row.
		return 0, err
	}
	return role.ID, nil
}

/**
 * USER DATA ACCESS METHODS
 * These methods handle core authentication and identity management.
 */

// CheckEmailExist is a high-performance helper to validate email uniqueness.
//
// Efficiency Note:
// Instead of retrieving the entire user object, it uses a SQL COUNT(*) query.
// This minimizes data transfer between the database server and the application.
//
// Return: 
// - true: Email is taken.
// - false: Email is available or an error occurred (count will be 0).
func CheckEmailExist(email string) bool {
	var count int64

	// Query: SELECT count(*) FROM users WHERE email = ?;
	// We bind the model here so GORM knows which table schema to target.
	db.DB.Model(&model.User{}).Where("email = ?", email).Count(&count)
	return count > 0
}

// CreateUser handles the physical insertion of a new user record.
//
// Database Constraints:
// It relies on the database schema to enforce integrity (e.g., NOT NULL, UNIQUE email).
// If any constraint is violated, GORM will return a non-nil error.
//
// Parameters:
// - user: A pointer to a populated User struct. GORM will auto-fill the 'ID', 
//   'CreatedAt', and 'UpdatedAt' fields if they are defined in the model.
func CreateUser(user *model.User) error {
	// Query: INSERT INTO users (...) VALUES (...);
	return db.DB.Create(user).Error
}

// GetUserByEmail is a critical function for the Authentication flow.
//
// Eager Loading Pattern (Preload):
// By using .Preload("Role"), GORM performs a "Join" or a secondary follow-up query 
// to fetch the User's role details in the same logical operation. 
// This prevents the "N+1 Problem" where the code would otherwise have to 
// query the role table separately for every user accessed.
//
// Error Handling:
// If no user matches the email, it returns (nil, gorm.ErrRecordNotFound).
func GetUserByEmail(email string) (*model.User, error) {
	var user model.User

	// Query: SELECT * FROM users WHERE email = ? LIMIT 1;
	// Followed by: SELECT * FROM roles WHERE id IN (user.role_id);
	err := db.DB.Where("email = ?", email).
		Preload("Role"). // Forces GORM to populate the user.Role nested struct.
		First(&user).Error

	if err != nil {
		// Wrapping or returning raw error for the service layer to handle (e.g., 404 or 401).
		return nil, err
	}

	return &user, nil
}
// GetUserProfileByID performs a LEFT JOIN between the 'User' table and the 'user_info' table.
// It aggregates basic account data and detailed profile data into a single UserProfile struct.
func GetUserProfileByID(id uint) (*model.UserProfile, error) {
	var profile model.UserProfile

	err := db.DB.Table("User").
		// Select specific columns to reduce bandwidth and protect sensitive data (like password hashes).
		Select("User.name, User.email, User.avatar_url, user_info.date_of_birth, user_info.gender, user_info.phone_number, user_info.address").
		// user_info might not exist yet for new users, so a LEFT JOIN ensures the User record is still returned.
		Joins("left join user_info on user_info.user_id = User.id").
		Where("User.id = ?", id).
		Scan(&profile).Error // Scan maps the join results directly into the profile struct fields.

	return &profile, err
}

// UpdateUserProfile manages a cross-table partial update for user account and profile data.
// 
// Transactional Design (ACID Properties):
// This function utilizes db.DB.Transaction to ensure Atomicity. Since we are modifying
// two separate tables ('User' and 'user_info'), the transaction prevents an "orphan update" 
// scenario where the name changes but the phone number fails, leaving the database in an 
// inconsistent state.
//
// Technical Constraint - Pointer-based Optionality:
// By using pointers (*string), this function checks for 'nil' to see if a field was 
// provided. 
// PRO TIP: This is a "Standard Partial Update" but suffers from the inability to 
// explicitly set a field to NULL via JSON, as both an omitted field and a null 
// field decode to 'nil' in standard Go JSON unmarshaling.
func UpdateUserProfile(id uint, p model.UserProfile) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {

		// Initialize a map to hold dynamic update fields for the 'User' table.
		// Maps are preferred over Structs in GORM updates because Structs ignore 
		// "Zero Values" (empty strings, 0, false), while Maps include everything present.
		userUpdates := map[string]interface{}{}

		// Dereference pointers only if they are not nil to prevent runtime panics.
		if p.Name != nil {
			userUpdates["name"] = *p.Name
		}
		if p.AvatarURL != nil {
			userUpdates["avatar_url"] = *p.AvatarURL
		}

		// Perform the primary table update only if at least one relevant field was provided.
		if len(userUpdates) > 0 {
			if err := tx.Model(&model.User{}).
				Where("id = ?", id).
				Updates(userUpdates).Error; err != nil {
				// Returning an error inside tx.Transaction triggers an automatic ROLLBACK.
				return err
			}
		}

		/* -------------------- user_info Table Logic -------------------- */
		
		// Attempt to locate the auxiliary profile record.
		var info model.UserInfo
		err := tx.Where("user_id = ?", id).First(&info).Error

		// Construct a secondary map for the extended profile attributes.
		infoUpdates := map[string]interface{}{}
		if p.DateOfBirth != nil { infoUpdates["date_of_birth"] = *p.DateOfBirth }
		if p.Gender != nil { infoUpdates["gender"] = *p.Gender }
		if p.PhoneNumber != nil { infoUpdates["phone_number"] = *p.PhoneNumber }
		if p.Address != nil { infoUpdates["address"] = *p.Address }

		// HANDLE MISSING PROFILE: If the user exists but has no user_info row yet.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Optimization: If no profile fields were provided, don't bother creating an empty row.
			if len(infoUpdates) == 0 {
				return nil
			}

			// Initialize a new UserInfo entity with the foreign key.
			newInfo := model.UserInfo{UserID: id}
			if p.DateOfBirth != nil { newInfo.DateOfBirth = *p.DateOfBirth }
			if p.Gender != nil { newInfo.Gender = *p.Gender }
			if p.PhoneNumber != nil { newInfo.PhoneNumber = *p.PhoneNumber }
			if p.Address != nil { newInfo.Address = *p.Address }

			// SQL: INSERT INTO user_info (...) VALUES (...);
			return tx.Create(&newInfo).Error
		}

		// NORMAL UPDATE: If the row exists, apply the changes.
		if len(infoUpdates) > 0 {
			// SQL: UPDATE user_info SET ... WHERE id = ...;
			return tx.Model(&info).Updates(infoUpdates).Error
		}

		return nil
	})
}

// UpdateUserProfilePatch represents the "Gold Standard" for RESTful PATCH implementation.
//
// The "Triple-State" Logic:
// This function solves the classic JSON ambiguity problem using a 'Set' flag:
// 1. Omitted: (Set=false) -> The database column remains untouched.
// 2. Explicit Null: (Set=true, Valid=false) -> The database column is set to NULL.
// 3. New Value: (Set=true, Valid=true) -> The database column is updated to the Value.
func UpdateUserProfilePatch(id uint, p model.UserProfilePatch) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {

		/* -------------------- Stage 1: User Table -------------------- */
		userUpdates := map[string]interface{}{}

		// Process Name field using 'Set' state detection.
		if p.Name.Set {
			if p.Name.Valid {
				userUpdates["name"] = p.Name.Value
			} else {
				// Explicitly nullify the field in the DB.
				userUpdates["name"] = nil 
			}
		}

		if p.AvatarURL.Set {
			if p.AvatarURL.Valid {
				userUpdates["avatar_url"] = p.AvatarURL.Value
			} else {
				userUpdates["avatar_url"] = nil
			}
		}

		// Execute update if the payload contained any 'User' table keys.
		if len(userUpdates) > 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", id).Updates(userUpdates).Error; err != nil {
				return err
			}
		}

		/* -------------------- Stage 2: user_info Table -------------------- */
		var info model.UserInfo
		err := tx.Where("user_id = ?", id).First(&info).Error

		// Construct the update map using the mapValue helper to handle Valid/Value logic.
		infoUpdates := map[string]interface{}{}

		if p.DateOfBirth.Set {
			infoUpdates["date_of_birth"] = mapValue(p.DateOfBirth)
		}
		if p.Gender.Set {
			infoUpdates["gender"] = mapValue(p.Gender)
		}
		if p.PhoneNumber.Set {
			infoUpdates["phone_number"] = mapValue(p.PhoneNumber)
		}
		if p.Address.Set {
			infoUpdates["address"] = mapValue(p.Address)
		}

		// CREATION LOGIC for user_info.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Check if any profile-related field was actually 'Set' in the request.
			shouldCreate := p.DateOfBirth.Set || p.Gender.Set || p.PhoneNumber.Set || p.Address.Set
			if !shouldCreate {
				// No data to act upon; exit the transaction successfully.
				return nil
			}

			newInfo := model.UserInfo{UserID: id}
			// Only populate the struct with valid string values. 
			// Fields that are 'Set' but not 'Valid' (null) will default to DB nulls upon Create.
			if p.DateOfBirth.Set && p.DateOfBirth.Valid { newInfo.DateOfBirth = p.DateOfBirth.Value }
			if p.Gender.Set && p.Gender.Valid { newInfo.Gender = p.Gender.Value }
			if p.PhoneNumber.Set && p.PhoneNumber.Valid { newInfo.PhoneNumber = p.PhoneNumber.Value }
			if p.Address.Set && p.Address.Valid { newInfo.Address = p.Address.Value }

			return tx.Create(&newInfo).Error
		}

		// UPDATE LOGIC: Apply the 'Triple-State' map to the existing record.
		if len(infoUpdates) > 0 {
			// tx.Model(&info).Updates(...) generates the efficient SQL SET clause.
			return tx.Model(&info).Updates(infoUpdates).Error
		}

		return nil
	})
}

// mapValue is a conceptual helper to clean up the code above (Logic explained in Patch comments).
func mapValue(field struct{Set bool; Valid bool; Value string}) interface{} {
	if field.Valid { return field.Value }
	return nil
}

// DeleteUserByID removes a user record from the system.
// It returns an error if the user does not exist (RowsAffected == 0).
func DeleteUserByID(id uint) error {
	result := db.DB.Delete(&model.User{}, id)

	if result.Error != nil {
		return result.Error
	}

	// Safety check: ensure that we actually deleted something.
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// UpdateUserRoleByID changes the permission level (Role) of a user.
// Common for administrative actions (e.g., promoting a student to instructor).
func UpdateUserRoleByID(userID uint, roleID uint) error {
	return db.DB.Model(&model.User{}).
		Where("id = ?", userID).
		Update("role_id", roleID).Error
}