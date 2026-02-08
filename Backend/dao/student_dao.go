package dao

import (
	"errors"
	"my-course-backend/db"
	"my-course-backend/model"
)

// GetRoleByName retrieves the role ID based on the role name.
func GetRoleByName(name string) (uint, error) {
	var role model.Role
	// Query the database and return the error directly if it fails
	err := db.DB.Where("role_name = ?", name).First(&role).Error
	if err != nil {
		return 0, err
	}
	return role.ID, nil
}

// CheckEmailExist checks if the given email already exists in the database.
func CheckEmailExist(email string) bool {
	var count int64
	db.DB.Model(&model.Student{}).Where("email = ?", email).Count(&count)
	return count > 0
}

// CreateStudent creates a new student record in the database.
func CreateStudent(student *model.Student) error {
	return db.DB.Create(student).Error
}

// GetStudentByEmail finds a student by email and preloads their Role information.
func GetStudentByEmail(email string) (*model.Student, error) {
	var student model.Student
	// Preload("Role") eagerly loads the associated Role data, which is often needed by the frontend.
	err := db.DB.Where("email = ?", email).Preload("Role").First(&student).Error
	if err != nil {
		return nil, err
	}
	return &student, nil
}

// DeleteStudentByID deletes a student record by their ID.
func DeleteStudentByID(id uint) error {
	// Perform the delete operation
	result := db.DB.Delete(&model.Student{}, id)
	
	if result.Error != nil {
		return result.Error
	}
	
	// If RowsAffected is 0, it means the ID does not exist in the database.
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	
	return nil
}