// src/App.tsx
import { Routes, Route, Navigate } from "react-router-dom";
import MainLayout from "./layouts/MainLayout";

// 引入刚才建的页面
import Login from "./pages/Login";
import Register from "./pages/Register";
import CourseList from "./pages/CourseList";
import StudentDashboard from "./pages/StudentDashboard";

function App() {
  return (
    <Routes>
      {/* 1. Public Routes - no need for MainLayout */}
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />

      {/* 2. Protected Routes - use MainLayout */}
      {/* The path="/" here means all child routes will have MainLayout's navigation bar */}
      <Route element={<MainLayout />}>
        {/*  */}
        <Route path="/" element={<Navigate to="/courses" replace />} />

        <Route path="/courses" element={<CourseList />} />
        <Route path="/dashboard" element={<StudentDashboard />} />
      </Route>

      {/* 404 Page - redirect to login for any unmatched route */}
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  );
}

export default App;
