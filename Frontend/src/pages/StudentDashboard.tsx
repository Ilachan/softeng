import Button from "../components/ui/Button";
import { Icons } from "../lib/icons";
import { useAuthStore } from "../store/authStore";
import { useNavigate, Outlet, NavLink } from "react-router-dom";
import BackgroundBlobs from "../components/ui/BackgroundBlobs";

const StudentDashboard = () => {
  const { logout } = useAuthStore();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  return (
    <div className="min-h-screen text-slate-800 relative">
      <BackgroundBlobs />

      {/* Navbar (Glass) */}
      <nav className="sticky top-0 z-50 border-b border-white/20 bg-white/70 backdrop-blur-xl">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-20 items-center">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-linear-to-tr from-violet-600 to-indigo-600 rounded-xl flex items-center justify-center text-white font-bold shadow-md">
                F
              </div>
              <span className="text-xl font-bold bg-clip-text text-transparent bg-linear-to-r from-violet-600 to-indigo-600">
                FitFlow
              </span>
            </div>
            <div className="flex items-center gap-2 bg-slate-100/50 p-1 rounded-xl">
              <NavLink
                to="browse"
                className={({ isActive }) =>
                  `px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
                    isActive
                      ? "bg-white text-indigo-600 shadow-sm"
                      : "text-slate-500 hover:text-slate-700"
                  }`
                }
              >
                Browse
              </NavLink>
              <NavLink
                to="my-schedule"
                className={({ isActive }) =>
                  `px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
                    isActive
                      ? "bg-white text-indigo-600 shadow-sm"
                      : "text-slate-500 hover:text-slate-700"
                  }`
                }
              >
                My Schedule
              </NavLink>
            </div>
            <Button
              variant="ghost"
              onClick={handleLogout}
              className="hidden sm:flex"
            >
              <Icons.Logout /> Logout
            </Button>
          </div>
        </div>
      </nav>

      <main className="max-w-6xl mx-auto px-4 py-12">
        <Outlet />
      </main>
    </div>
  );
};

export default StudentDashboard;
