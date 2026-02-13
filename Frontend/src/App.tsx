import { Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Browse from "./pages/Browse";
import MySchedule from "./pages/MySchedule";
import MainLayout from "./layouts/MainLayout";
import AuthGuard from "./components/AuthGuard";

function App() {
  return (
    <main>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />

        {/* Protected Routes */}
        <Route element={<AuthGuard />}>
          <Route path="/dashboard" element={<MainLayout />}>
            <Route index element={<Navigate to="browse" replace />} />
            <Route path="browse" element={<Browse />} />
            <Route path="my-schedule" element={<MySchedule />} />
          </Route>
          {/* Default redirect for authenticated users targeting root */}
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
        </Route>

        {/* Fallback for unknown routes */}
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </main>
  );
}

export default App;
