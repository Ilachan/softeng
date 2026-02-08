package api

import (
	"net/http"
	"strconv"
	
	// Ensure these match your go.mod module name
	"my-course-backend/model"
	"my-course-backend/service"

	"github.com/gin-gonic/gin"
)

// Register handles user registration
func Register(c *gin.Context) {
	var input model.RegisterInput

	// 1. Bind and validate parameters (Gin validates based on struct binding tags)
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Call Service to register
	if err := service.RegisterUser(input); err != nil {
		// If email already exists, return 409 Conflict; otherwise return 500
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Student registered successfully"})
}

// Login handles user login
func Login(c *gin.Context) {
	var input model.LoginInput

	// 1. Bind parameters
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Call Service to login
	token, err := service.LoginUser(input)
	if err != nil {
		// Return 401 Unauthorized for generic errors to prevent username enumeration
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// 3. Login successful, return Token
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}

// DeleteUser handles user deletion
func DeleteUser(c *gin.Context) {
	// 1. Get 'id' parameter from URL (string type)
	idStr := c.Param("id")

	// 2. Convert string to uint
	// ParseUint(string, base, bitSize)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// 3. Call Service
	if err := service.RemoveUser(uint(id)); err != nil {
		// Return 404 if user not found, otherwise 500
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 4. Return success
	c.JSON(http.StatusOK, gin.H{"message": "User and related data deleted successfully"})
}