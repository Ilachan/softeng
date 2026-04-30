# Sprint 4 Report: FitFlow Team

**Team Members:**

- **Frontend:** Forrest Yan Sun, Ila Adhikari
- **Backend:** Qing Li, Yingzhu Chen

**Project Links:**

- 🔗 **GitHub Repository:** [https://github.com/Ilachan/FitFlow](https://github.com/Ilachan/FitFlow)
- 📺 **Demo Video:** [[https://youtu.be/ED_eJzzcMJ4](https://youtu.be/ED_eJzzcMJ4)](https://youtu.be/UtAVc_wOMJQ)
  - Note: As this is the final sprint, the demonstration video includes both the updates for Sprint 4 and a comprehensive summary of the entire project.

---

<!-- Content tabs / quick links -->

**Jump to:** [Detailed Work](#1-detail-work-completed-in-sprint-4) | [Frontend Testing](#2-frontend-testing-summary) | [Backend Testing](#3-backend-testing-summary) | [API Doc](#4-api-documentation-updates) | [Summary](#5-summary)

---

## 1) Detail Work Completed in Sprint 4

### 1.1 User Stories & Features

#### 1) Return All Classes (Remove Pagination)

The `GET /classes` endpoint previously returned a paginated subset of classes. It now returns all classes in a single response.

**Acceptance Criteria:**

- All classes are returned without pagination parameters
- Response structure remains consistent with previous paginated response
- Supports optional filters (category and day of week)

**Backend Work Completed**

- Removed `page` and `pageSize` query parameters from the handler
- Updated `ListClassesPaged` in the service and DAO layers accordingly
- Response structure remains the same; the full class list is returned in one call

**Frontend Work Completed**

- Updated `Browse.tsx` to fetch all classes without pagination logic
- Simplified class list rendering to display all returned classes
- Enhanced UI with filtering UI controls for category and day of week

---

#### 2) Instructor Profile (Instructor Table)

Designed and implemented a dedicated `Instructor` table to store instructor-specific profile data, separate from the general `User` table. This enables rich instructor profiles with extended metadata (bio, qualifications, etc.).

**Acceptance Criteria:**

- Instructors have a separate profile table with bio information
- Instructor-course relationships are maintained via name-based association
- Instructor data persists and can be retrieved via API

**Database Schema**

**New table — `Instructor`:**
| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER | Primary key |
| `user_id` | INTEGER | FK → User (role_id = 4 only) |
| `bio` | TEXT | Instructor biography |

**Backend Work Completed**

- Added `model/instructor.go` with the `Instructor` struct
- Auto-migration creates the table on startup
- The `Course` table's `instructor_id` (integer FK) column was replaced with `instructor` (TEXT), storing the instructor's display name directly
- A one-time migration (`migrateCourseInstructorToName`) backfills existing rows with the correct name
- Instructor ownership checks in all service methods now use name-based comparison instead of ID-based
- Implemented instructor course listing endpoints: `GET /instructor/courses`
- Implemented instructor enrollment management APIs

**Frontend Work Completed**

- Created dedicated `InstructorDashboard.tsx` page for instructor role
- Implemented `InstructorProfile.tsx` to display and edit instructor bio
- Added instructor course list view with action buttons
- Integrated instructor dashboard navigation in sidebar/navbar
- Created test coverage for instructor dashboard and profile pages

---

#### 3) Class Filter (Category & Day of Week)

Students can now filter the class list by category and/or day of the week on the class exploration page.

**Acceptance Criteria:**

- Students can filter by category (single select)
- Students can filter by day of week (single select)
- Filters are optional and can be combined
- Filter options are dynamically populated from available classes

**Updated endpoint:** `GET /classes?category=Yoga&weekday=1`

| Query Param | Type   | Description                                           |
| ----------- | ------ | ----------------------------------------------------- |
| `category`  | string | Filter by class category (e.g. `Yoga`, `Pilates`)     |
| `weekday`   | int    | Filter by day of week (`0` = Sunday … `6` = Saturday) |

**Backend Work Completed**

- Both filters are optional and can be combined
- Added `GET /classes/categories` endpoint to return all distinct category values for populating the filter dropdown
- DAO, service, and API layers all updated to pass filters through the full call chain
- Implemented filter logic in `ListClasses` service method

**Frontend Work Completed**

- Enhanced `Browse.tsx` with category and weekday filter dropdowns
- Integrated filter state management
- Real-time filter application without page reload
- Updated API calls to include filter parameters
- Added test coverage for filter functionality

---


#### 4) Instructor Adds Standby Student

Instructors can now add a student as a standby student for their courses.

**Acceptance Criteria:**

- Instructor can add any student as a standby student to a course they teach via a backend API
- The system prevents duplicate standbys and enforces maximum standby capacity if applicable
- Standby students can be retrieved via a dedicated API

**Backend Work Completed**

- Added `POST /manager/users/{user_id}/enrollments` endpoint to allow instructors to add students to a course’s standby list
- Updated the data model to track standby status and validate standby capacity

---

#### 5) Instructor Removes Standby Student

Instructors can remove a student from the standby list of their course to dynamically manage the standby queue.

**Acceptance Criteria:**

- Instructor can remove a specific standby student from their course via a backend API
- Once removed, the student no longer occupies a standby spot
- The API performs permission checks and provides clear feedback

**Backend Work Completed**

- Added `DELETE /instructor/courses/{course_id}/enrollments/{user_id}` endpoint to allow instructors to remove a student from the standby list
- Implemented permission, status validation, and appropriate response handling


## 2) Frontend Testing Summary

### 2.1 Unit Tests

The following unit test files provide coverage for core frontend functionality:

- **`src/lib/api.test.ts`** - API integration and HTTP request handling
- **`src/lib/validation.test.ts`** - Form validation utilities and email/password strength checks
- **`src/store/authStore.test.ts`** - Authentication state management and token handling
- **`src/pages/Login.test.tsx`** - Login form and authentication flow
- **`src/pages/Register.test.tsx`** - Registration form and validation feedback
- **`src/pages/Profile.test.tsx`** - User profile display and basic profile operations
- **`src/pages/Browse.test.tsx`** - Class listing, filtering by category and day of week
- **`src/pages/InstructorDashboard.test.tsx`** - Instructor dashboard with course list and enrollment management
- **`src/pages/InstructorProfile.test.tsx`** - Instructor profile bio display and edit functionality

### 2.2 Cypress E2E Tests

The following Cypress tests provide end-to-end test coverage for critical user workflows:

- **`cypress/e2e/auth-smoke.cy.ts`** - Basic authentication flows (login/logout)
- **`cypress/e2e/auth-guard.cy.ts`** - Route protection and unauthorized access handling
- **`cypress/e2e/login-validation.cy.ts`** - Login form validation and error states
- **`cypress/e2e/register-validation.cy.ts`** - Registration form validation and acceptance

---

## 3) Backend Testing Summary

### 3.1 Unit Tests

**Overall Status:** PASS

**Test Packages:**

- `routes/class_routes_test.go` - Class-related API endpoints (list, get, register, drop, create, update, delete)
- `routes/instructor_routes_test.go` - Instructor API endpoints (list courses, list enrollments, update enrollment status)
- `routes/manager_routes_test.go` - Manager API endpoints (create/update/delete classes, user management)
- `routes/supermanager_invite_routes_test.go` - SuperManager invite code generation

**Coverage:**

Notable test coverage includes:

- Class pagination removal and filtering by category/day of week
- Instructor course listing and enrollment retrieval
- Instructor enrollment status updates (attended/missed/enrolled)
- Class capacity validation and enrollment management
- Time conflict detection during registration
- Manager and SuperManager permission checks

**Test Execution:**

- Runtime: approximately 3.7-4.0 seconds for the full suite
- All tests verify both positive and negative paths (authorization, validation, edge cases)

---

## 4) API Documentation Updates

### Class Management

**List All Classes (with optional filters)**

- `GET /classes` - List all classes
  - Query params: `category` (optional), `weekday` (optional, 0-6)
  - Returns: array of all classes matching filters

**Get Class Categories**

- `GET /classes/categories` - Get distinct category values
  - Returns: array of category strings

**Get Class Details**

- `GET /classes/:id` - Get a single class by ID
  - Returns: class object with details

**List Class Enrollments**

- `GET /classes/:id/enrollments` - List all students enrolled in a class
  - Returns: array of enrollment objects (manager/instructor only)

**Student Register for Class**

- `POST /classes/register` - Enroll a student in a class
  - Body: `{ courseID }`
  - Validation: enrollment window, capacity, conflicts, duplicates
  - Returns: enrollment confirmation

**Student Drop Class**

- `POST /classes/drop` - Remove student from a class
  - Body: `{ courseID }`
  - Returns: success message

**Manager Create Class**

- `POST /classes` - Create a new class
  - Role required: Manager (role_id = 3) or SuperManager (role_id = 2)
  - Body: class creation data (name, description, category, capacity, time, instructor, etc.)
  - Returns: 201 created class object

**Manager Update Class**

- `PUT /classes/:id` - Update class details
  - Role required: Manager (role_id = 3) or SuperManager (role_id = 2)
  - Body: fields to update
  - Returns: 200 updated class object

**Manager Delete Class**

- `DELETE /classes/:id` - Delete a class
  - Role required: Manager (role_id = 3) or SuperManager (role_id = 2)
  - Returns: 200 success message

### Instructor Management

**List Instructor's Courses**

- `GET /instructor/courses` - List all courses taught by authenticated instructor
  - Auth required: role_id = 4 (Instructor)
  - Returns: array of course objects

**List Enrollments for Instructor's Course**

- `GET /instructor/courses/:id/enrollments` - List all students enrolled in instructor's course
  - Auth required: role_id = 4 (Instructor)
  - Path param: `:id` (course ID)
  - Returns: array of enrollment objects with student details

**Instructor Add Student to Class**

- `POST /instructor/courses/:id/enrollments` - Manually add a student to instructor's course
  - Auth required: role_id = 4 (Instructor)
  - Path param: `:id` (course ID)
  - Body: `{ userID }`
  - Validation: permission check, enrollment duplicate check
  - Returns: 201 created enrollment

**Instructor Update Enrollment Status**

- `PATCH /instructor/courses/:id/enrollments` - Update student attendance status
  - Auth required: role_id = 4 (Instructor)
  - Path param: `:id` (course ID)
  - Body: `{ userID, status }` where status is one of: "enrolled", "attended", "missed"
  - Returns: 200 updated enrollment object

**Instructor Remove Student from Class**

- `DELETE /instructor/courses/:id/enrollments/:user_id` - Remove a student from instructor's course
  - Auth required: role_id = 4 (Instructor)
  - Path params: `:id` (course ID), `:user_id` (student ID)
  - Returns: 200 success message

---

## 5) Summary

**Sprint 4 Achievements:**

This sprint focused on enhancing the class discovery and instructor management features:

1. **Removed pagination** from the class listing endpoint to provide students with a complete view of all available classes
2. **Implemented dedicated instructor profiles** with a separate `Instructor` table for storing bio and extended profile information
3. **Added class filtering** by category and day of week, allowing students to refine their class search
4. **Expanded instructor capabilities** with new APIs for managing course enrollments, including walk-in registrations and attendance tracking
5. **Maintained comprehensive test coverage** across both frontend unit tests and Cypress E2E tests
6. **Backend unit tests pass** with coverage of all new instructor endpoints and filtering logic

**Key Files Modified:**

- Backend: `api/instructor_api.go`, `routes/routes.go`, `dao/class_dao.go`, `service/class_service.go`, `service/instructor_service.go`, `model/instructor.go`
- Frontend: `Browse.tsx`, `InstructorDashboard.tsx`, `InstructorProfile.tsx`, `Navbar.tsx`, `CourseDetailsModal.tsx`, plus supporting components and test files
- Tests: Multiple new test cases in `routes/*_test.go`, and frontend unit + E2E test updates
