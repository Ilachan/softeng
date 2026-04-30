package model

/**
 * ============================================================================================
 * DATA MODEL: Instructor
 * ============================================================================================
 *
 * @description:
 * The Instructor struct represents the specialized profile data for teaching staff.
 * While basic identity data (email, password) resides in the 'User' model, this model
 * encapsulates professional-specific attributes like biographies and teaching names.
 *
 * @architecture_design:
 * This model follows a "Separation of Identity and Profile" pattern.
 * - Identity: Managed in the 'User' table.
 * - Profile: Managed in this 'Instructor' table.
 *
 * @role_specification:
 * By system convention, an Instructor record should only exist if the associated User
 * possesses a role_id of 4 (Instructor Role). This is enforced at the service layer
 * but supported here by the UserID foreign key.
 *
 * @relationship_integrity:
 * - One-to-One: Each Instructor record corresponds to exactly one User record.
 * - Cascading: Typically, if a User is deleted, the Instructor profile should follow.
 * ============================================================================================
 */
type Instructor struct {
	/* * @field: ID
	 * @type: uint (unsigned integer)
	 * @db_constraints: primaryKey, autoIncrement
	 * @description: The unique surrogate key for the Instructor table.
	 * It allows for independent indexing regardless of the User ID.
	 */
	ID uint `gorm:"primaryKey;autoIncrement;column:id" json:"id"`

	/* * @field: UserID
	 * @type: uint
	 * @db_constraints: column:user_id, not null, uniqueIndex
	 * @description: This field acts as the bridge between the Instructor and User tables.
	 * The 'uniqueIndex' is critical as it prevents a single user from having multiple
	 * instructor profiles, maintaining data cleanliness.
	 */
	UserID uint `gorm:"column:user_id;not null;uniqueIndex" json:"user_id"`

	/* * @field: Name
	 * @type: string
	 * @description: The professional display name. This may differ from the 'User.Name'
	 * used for billing or internal account management. It is what students see.
	 */
	Name string `gorm:"column:name" json:"name"`

	/* * @field: Bio
	 * @type: string
	 * @description: A textual description of the instructor's background, certifications,
	 * and expertise. In the database, this typically maps to a TEXT or VARCHAR(MAX) field.
	 */
	Bio string `gorm:"column:bio" json:"bio"`

	/* * @field: User
	 * @type: User (Struct)
	 * @orm_logic: foreignKey:UserID
	 * @description: An instance of the User struct. GORM uses this metadata to perform
	 * 'Preload' operations (Eager Loading), joining the User table during the fetch.
	 * This allows the API to return user-specific data (like Email or Avatar)
	 * alongside the bio in a single nested JSON object.
	 */
	User User `gorm:"foreignKey:UserID" json:"user"`
}

/**
 * ============================================================================================
 * ORM METHOD: TableName
 * ============================================================================================
 *
 * @description:
 * This method satisfies the GORM Tabler interface. By default, GORM would look for
 * a table named "instructors" (lowercase plural).
 *
 * @rationale:
 * We explicitly override this to return "Instructor" to match the specific CamelCase
 * and singular naming convention used in the legacy SQL database schema.
 *
 * @return: string - The exact name of the table in the database.
 * ============================================================================================
 */
func (Instructor) TableName() string {
	return "Instructor"
}

/* * ============================================================================================
 * END OF MODEL DEFINITION
 * ============================================================================================
 * [Reserved for additional documentation or future-proofing tags]
 * - Added support for Instructor-User mapping.
 * - Validated JSON tags for API compatibility.
 * - Ensured GORM tags align with database schema constraints.
 * - Prepared for service layer enforcement of role-based integrity.
 * ============================================================================================

 */
