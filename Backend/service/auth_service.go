package service

import (
	"errors"
	"my-course-backend/dao"
	"my-course-backend/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Secret key for JWT signing.
// WARNING: In a production environment, store this in environment variables (e.g., os.Getenv("JWT_SECRET")), DO NOT hardcode it here.
var jwtSecret = []byte("my_super_secret_key_2026")

// ==========================================
// 1. Registration Logic (RegisterUser)
// ==========================================
func RegisterUser(input model.RegisterInput) error {
	// 1. Validate email uniqueness
	// Check if the email is already registered in the database.
	if dao.CheckEmailExist(input.Email) {
		return errors.New("email already exists")
	}

	// 2. Hash the password (Bcrypt)
	// Never store plain-text passwords. Bcrypt handles salting and hashing automatically.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 3. Retrieve the "Student" Role ID
	// By default, all new users are assigned the "Student" role.
	roleID, err := dao.GetRoleByName("Student")
	if err != nil {
		return errors.New("role 'Student' not found in database")
	}

	// 4. Construct the Student object
	student := model.Student{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword), // Store the hashed password, not the raw one
		RoleID:   roleID,
	}

	// 5. Persist to database
	return dao.CreateStudent(&student)
}

// ==========================================
// 2. Login Logic (LoginUser)
// ==========================================
func LoginUser(input model.LoginInput) (string, error) {
	// 1. Find user by email
	student, err := dao.GetStudentByEmail(input.Email)
	if err != nil {
		return "", errors.New("user not found")
	}

	// 2. Verify Password
	// Compare the stored hash (from DB) with the input password (plain text).
	err = bcrypt.CompareHashAndPassword([]byte(student.Password), []byte(input.Password))
	if err != nil {
		return "", errors.New("invalid password")
	}

	// 3. Generate JWT Token
	// Create a new token object, specifying the signing method (HS256) and the claims (payload).
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":      student.ID,     // User ID
		"email":   student.Email,  // User Email
		"role_id": student.RoleID, // Role ID (used for frontend permission checks)
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Expiration time: 24 hours from now
	})

	// 4. Sign the token
	// Sign the token using the secret key to generate the final token string.
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// RemoveUser handles the business logic for deleting a user.
func RemoveUser(id uint) error {
	return dao.DeleteStudentByID(id)
}