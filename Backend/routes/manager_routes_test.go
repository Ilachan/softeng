package routes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"my-course-backend/db"
	"my-course-backend/model"
	"my-course-backend/routes"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func setupManagerRouteTestDB(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:manager_route_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := testDB.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	if err := testDB.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.UserInfo{},
		&model.Course{},
		&model.Enrollment{},
		&model.UserDailyActivity{},
		&model.ManagerInviteCode{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	db.DB = testDB
}

func seedRole(t *testing.T, id uint, name string) {
	t.Helper()
	if err := db.DB.Create(&model.Role{ID: id, RoleName: name}).Error; err != nil {
		t.Fatalf("failed to seed role: %v", err)
	}
}

func seedCourseForUpdate(t *testing.T) model.Course {
	t.Helper()

	startTime, _ := model.ParseTimeOnly("09:00")
	endTime, _ := model.ParseTimeOnly("10:00")

	c := model.Course{
		CourseName:  "Old Name",
		CourseCode:  "OLD-001",
		Description: "old desc",
		StartTime:   startTime,
		EndTime:     endTime,
		Capacity:    10,
		Duration:    60,
		Category:    "OldCat",
		Weekday:     "Monday",
	}
	if err := db.DB.Create(&c).Error; err != nil {
		t.Fatalf("failed to seed course: %v", err)
	}
	return c
}

// IMPORTANT: 你的 service/auth_service.go 里 jwtSecret 是硬编码的 []byte("my_super_secret_key_2026")。
// 这里为了让 role 校验通过，需要生成同样 secret 的 token。
var jwtSecretForTests = []byte("my_super_secret_key_2026")

func makeToken(t *testing.T, userID uint, roleID uint) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":      userID,
		"email":   "test@example.com",
		"role_id": roleID,
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	})

	s, err := token.SignedString(jwtSecretForTests)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return s
}

func TestManagerCanUpdateClass(t *testing.T) {
	setupManagerRouteTestDB(t)

	// seed roles: 3=Manager
	seedRole(t, 3, "Manager")

	// seed a course to update
	course := seedCourseForUpdate(t)

	r := routes.SetupRouter()

	body := map[string]any{
		"name":        "New Name",
		"course_code": "NEW-001",
		"description": "new desc",
		"start_time":  "08:00",
		"end_time":    "09:00",
		"capacity":    15,
		"duration":    45,
		"category":    "NewCat",
		"weekday":     "Tuesday",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/classes/%d", course.ID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeToken(t, 999, 3)) // role_id=3 manager

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	// verify DB updated
	var updated model.Course
	if err := db.DB.First(&updated, course.ID).Error; err != nil {
		t.Fatalf("failed to fetch updated course: %v", err)
	}
	if updated.CourseName != "New Name" || updated.CourseCode != "NEW-001" || updated.Capacity != 15 {
		t.Fatalf("course not updated as expected: %+v", updated)
	}
}

func TestStudentCannotUpdateClass(t *testing.T) {
	setupManagerRouteTestDB(t)

	// seed roles: 1=Student
	seedRole(t, 1, "Student")

	course := seedCourseForUpdate(t)
	r := routes.SetupRouter()

	body := map[string]any{
		"name":        "Should Fail",
		"course_code": "FAIL-001",
		"description": "x",
		"start_time":  "08:00",
		"end_time":    "09:00",
		"capacity":    15,
		"duration":    45,
		"category":    "NewCat",
		"weekday":     "Tuesday",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/classes/%d", course.ID), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeToken(t, 1000, 1)) // role_id=1 student

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d, body=%s", w.Code, w.Body.String())
	}
}

// ─── Helper: setup with ClassSession support ─────────────────────

func setupManagerTestDBWithSessions(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:mgr_session_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := testDB.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}
	if err := testDB.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.UserInfo{},
		&model.Course{},
		&model.ClassSession{},
		&model.Enrollment{},
		&model.UserDailyActivity{},
		&model.ManagerInviteCode{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	db.DB = testDB
}

func seedManagerCourseWithSession(t *testing.T, name string, capacity int) model.Course {
	t.Helper()
	startTime, _ := model.ParseTimeOnly("09:00")
	endTime, _ := model.ParseTimeOnly("10:00")

	course := model.Course{
		CourseName: name,
		CourseCode: fmt.Sprintf("MGR-%d", time.Now().UnixNano()),
		Capacity:   capacity,
		Category:   "Fitness",
		StartTime:  startTime,
		EndTime:    endTime,
		Weekday:    "Monday",
		Duration:   60,
	}
	if err := db.DB.Create(&course).Error; err != nil {
		t.Fatalf("failed to seed course: %v", err)
	}

	sessionDate := time.Now().AddDate(0, 0, 1)
	session := model.ClassSession{
		CourseID:    course.ID,
		SessionDate: sessionDate.Format("2006-01-02"),
		StartAt:     time.Date(sessionDate.Year(), sessionDate.Month(), sessionDate.Day(), 9, 0, 0, 0, time.UTC),
		EndAt:       time.Date(sessionDate.Year(), sessionDate.Month(), sessionDate.Day(), 10, 0, 0, 0, time.UTC),
		Status:      "scheduled",
		Capacity:    capacity,
	}
	if err := db.DB.Create(&session).Error; err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
	return course
}

func seedManagerEnrollment(t *testing.T, userID, courseID uint) model.Enrollment {
	t.Helper()
	var session model.ClassSession
	if err := db.DB.Where("course_id = ? AND status = ?", courseID, "scheduled").
		Order("session_date ASC").First(&session).Error; err != nil {
		t.Fatalf("failed to find session: %v", err)
	}

	enrollment := model.Enrollment{
		UserID:     userID,
		CourseID:   courseID,
		SessionID:  &session.ID,
		Status:     model.EnrollmentStatusEnrolled,
		EnrollTime: time.Now(),
	}
	if err := db.DB.Create(&enrollment).Error; err != nil {
		t.Fatalf("failed to seed enrollment: %v", err)
	}
	return enrollment
}

// ─── GET /manager/users ──────────────────────────────────────────

func TestManagerListUsers_OK(t *testing.T) {
	setupManagerTestDBWithSessions(t)
	seedRole(t, 3, "Manager")
	seedRole(t, 1, "Student")

	// Create some users
	for i := 0; i < 3; i++ {
		db.DB.Create(&model.User{
			Name:     fmt.Sprintf("User %d", i),
			Email:    fmt.Sprintf("user-%d-%d@example.com", i, time.Now().UnixNano()),
			Password: "pass",
			RoleID:   1,
		})
	}

	token := makeToken(t, 999, 3) // manager
	r := routes.SetupRouter()

	req := httptest.NewRequest(http.MethodGet, "/manager/users?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Users      []model.User `json:"users"`
		Total      int64        `json:"total"`
		Page       int          `json:"page"`
		Limit      int          `json:"limit"`
		TotalPages int          `json:"total_pages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total 3, got %d", resp.Total)
	}
	if len(resp.Users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(resp.Users))
	}
}

func TestManagerListUsers_Forbidden_NonManager(t *testing.T) {
	setupManagerTestDBWithSessions(t)
	seedRole(t, 1, "Student")

	token := makeToken(t, 999, 1) // non-manager role
	r := routes.SetupRouter()

	req := httptest.NewRequest(http.MethodGet, "/manager/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── POST /manager/users/:id/enrollments ─────────────────────────

// func TestManagerAddUserEnrollment_OK(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&user)

// 	course := seedManagerCourseWithSession(t, "Yoga", 10)
// 	token := makeToken(t, 999, 3) // manager
// 	r := routes.SetupRouter()

// 	body, _ := json.Marshal(map[string]uint{"course_id": course.ID})
// 	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/manager/users/%d/enrollments", user.ID), bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusCreated {
// 		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
// 	}

// 	// Verify enrollment has session_id set
// 	var enrollment model.Enrollment
// 	if err := db.DB.Where("user_id = ? AND course_id = ?", user.ID, course.ID).First(&enrollment).Error; err != nil {
// 		t.Fatalf("enrollment not found in DB: %v", err)
// 	}
// 	if enrollment.SessionID == nil {
// 		t.Fatal("expected session_id to be set, got nil")
// 	}
// 	if enrollment.Status != model.EnrollmentStatusEnrolled {
// 		t.Fatalf("expected status 'enrolled', got '%s'", enrollment.Status)
// 	}
// }

// func TestManagerAddUserEnrollment_Duplicate(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&user)

// 	course := seedManagerCourseWithSession(t, "Spin", 10)
// 	seedManagerEnrollment(t, user.ID, course.ID)

// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	body, _ := json.Marshal(map[string]uint{"course_id": course.ID})
// 	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/manager/users/%d/enrollments", user.ID), bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusConflict {
// 		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
// 	}
// }

// func TestManagerAddUserEnrollment_ClassFull(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	course := seedManagerCourseWithSession(t, "Full Class", 1) // capacity=1

// 	// Fill the spot
// 	existing := model.User{Name: "Existing", Email: fmt.Sprintf("e-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&existing)
// 	seedManagerEnrollment(t, existing.ID, course.ID)

// 	// Try to add another
// 	newUser := model.User{Name: "New User", Email: fmt.Sprintf("n-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&newUser)

// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	body, _ := json.Marshal(map[string]uint{"course_id": course.ID})
// 	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/manager/users/%d/enrollments", newUser.ID), bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusConflict {
// 		t.Fatalf("expected 409 (class is full), got %d: %s", w.Code, w.Body.String())
// 	}
// }

// func TestManagerAddUserEnrollment_UserNotFound(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")

// 	course := seedManagerCourseWithSession(t, "Yoga", 10)
// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	body, _ := json.Marshal(map[string]uint{"course_id": course.ID})
// 	req := httptest.NewRequest(http.MethodPost, "/manager/users/9999/enrollments", bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusNotFound {
// 		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
// 	}
// }

// ─── DELETE /manager/users/:id/enrollments/:course_id ────────────

// func TestManagerDeleteUserEnrollment_OK(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&user)

// 	course := seedManagerCourseWithSession(t, "Rowing", 10)
// 	seedManagerEnrollment(t, user.ID, course.ID)

// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	path := fmt.Sprintf("/manager/users/%d/enrollments/%d", user.ID, course.ID)
// 	req := httptest.NewRequest(http.MethodDelete, path, nil)
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
// 	}

// 	// Verify deleted
// 	var count int64
// 	db.DB.Model(&model.Enrollment{}).Where("user_id = ? AND course_id = ?", user.ID, course.ID).Count(&count)
// 	if count != 0 {
// 		t.Fatalf("expected enrollment to be deleted, but found %d", count)
// 	}
// }

// func TestManagerDeleteUserEnrollment_NotFound(t *testing.T) {
// 	setupManagerTestDBWithSessions(t)
// 	seedRole(t, 3, "Manager")
// 	seedRole(t, 1, "Student")

// 	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
// 	db.DB.Create(&user)

// 	token := makeToken(t, 999, 3)
// 	r := routes.SetupRouter()

// 	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/manager/users/%d/enrollments/9999", user.ID), nil)
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, req)

// 	if w.Code != http.StatusNotFound {
// 		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
// 	}
// }

// // ─── GET /manager/users/:id/enrollments ──────────────────────────

func TestManagerListUserEnrollments_OK(t *testing.T) {
	setupManagerTestDBWithSessions(t)
	seedRole(t, 3, "Manager")
	seedRole(t, 1, "Student")

	user := model.User{Name: "Test User", Email: fmt.Sprintf("u-%d@example.com", time.Now().UnixNano()), Password: "pass", RoleID: 1}
	db.DB.Create(&user)

	course1 := seedManagerCourseWithSession(t, "Course A", 10)
	course2 := seedManagerCourseWithSession(t, "Course B", 10)
	seedManagerEnrollment(t, user.ID, course1.ID)
	seedManagerEnrollment(t, user.ID, course2.ID)

	token := makeToken(t, 999, 3)
	r := routes.SetupRouter()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/manager/users/%d/enrollments", user.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Enrollments []model.Enrollment `json:"enrollments"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Enrollments) != 2 {
		t.Fatalf("expected 2 enrollments, got %d", len(resp.Enrollments))
	}
}

func TestManagerListUserEnrollments_UserNotFound(t *testing.T) {
	setupManagerTestDBWithSessions(t)
	seedRole(t, 3, "Manager")

	token := makeToken(t, 999, 3)
	r := routes.SetupRouter()

	req := httptest.NewRequest(http.MethodGet, "/manager/users/9999/enrollments", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
