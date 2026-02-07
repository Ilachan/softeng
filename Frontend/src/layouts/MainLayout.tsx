// src/layouts/MainLayout.tsx
import { Outlet, Link } from "react-router-dom";

const MainLayout = () => {
  return (
    <div className="min-h-screen bg-gray-50">
      {/* nav bar */}
      <nav className="bg-white shadow-sm p-4 flex gap-4">
        <Link to="/courses" className="font-bold text-blue-600">
          Course System
        </Link>
        <div className="flex gap-4">
          <Link to="/courses" className="hover:text-blue-500">
            All Courses
          </Link>
          <Link to="/dashboard" className="hover:text-blue-500">
            My Dashboard
          </Link>
          {/* <Link to="/admin" className="text-red-500">
            Admin
          </Link> */}
        </div>
        <div className="ml-auto">
          <Link to="/login">Logout</Link>
        </div>
      </nav>

      {/* pages */}
      <main className="p-8">
        <Outlet />
      </main>
    </div>
  );
};

export default MainLayout;
