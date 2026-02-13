import { Outlet } from "react-router-dom";
import BackgroundBlobs from "../components/ui/BackgroundBlobs";
import Navbar from "../components/Navbar";

const MainLayout = () => {
  return (
    <div className="min-h-screen text-slate-800 relative">
      <BackgroundBlobs />
      <Navbar />
      <main className="max-w-6xl mx-auto px-4 py-12">
        <Outlet />
      </main>
    </div>
  );
};

export default MainLayout;
