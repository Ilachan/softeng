package main

import (
	"log" // [Fixed] Added missing import for log.Printf
	
	// Ensure these match your go.mod module name
	"my-course-backend/db"
	"my-course-backend/model"
	"my-course-backend/routes" 
)

func main() {
	// 1. Initialize Database
	db.InitDB()

	// 2. Auto Migrate Table Structures
	db.DB.AutoMigrate(&model.Role{}, &model.Student{})

	// 3. Seed Initial Data
	seedRoles()

	// 4. Initialize Router (Call function from routes package)
	r := routes.SetupRouter()

	// 5. Start Server
	r.Run(":8080")
}

// seedRoles ensures default roles exist in the database
func seedRoles() {
	roles := []string{"Admin", "Teacher", "Student"}
	for _, roleName := range roles {
		var role model.Role
		// Check if role exists, if not, create it
		if err := db.DB.Where("role_name = ?", roleName).First(&role).Error; err != nil {
			db.DB.Create(&model.Role{RoleName: roleName})
			log.Printf("Created role: %s", roleName)
		}
	}
}