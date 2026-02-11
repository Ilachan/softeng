import React, { useState } from "react";
import Button from "../components/ui/Button";
import { Icons } from "../lib/icons";
import Browse from "./Browse";
import MySchedule from "./MySchedule";

const Dashboard = ({ onLogout }: { onLogout: () => void }) => {
  const [activeTab, setActiveTab] = useState("browse");

  return (
    <div className="min-h-screen text-slate-800 relative">
      <BackgroundBlobs />

      {/* Navbar (Glass) */}
      <nav className="sticky top-0 z-50 border-b border-white/20 bg-white/70 backdrop-blur-xl">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-20 items-center">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-tr from-violet-600 to-indigo-600 rounded-xl flex items-center justify-center text-white font-bold shadow-md">
                F
              </div>
              <span className="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-violet-600 to-indigo-600">
                FitFlow
              </span>
            </div>
            <div className="flex items-center gap-2 bg-slate-100/50 p-1 rounded-xl">
              <button
                onClick={() => setActiveTab("browse")}
                className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
                  activeTab === "browse"
                    ? "bg-white text-indigo-600 shadow-sm"
                    : "text-slate-500 hover:text-slate-700"
                }`}
              >
                Browse
              </button>
              <button
                onClick={() => setActiveTab("my")}
                className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
                  activeTab === "my"
                    ? "bg-white text-indigo-600 shadow-sm"
                    : "text-slate-500 hover:text-slate-700"
                }`}
              >
                My Schedule
              </button>
            </div>
            <Button
              variant="ghost"
              onClick={onLogout}
              className="hidden sm:flex"
            >
              <Icons.Logout /> Logout
            </Button>
          </div>
        </div>
      </nav>

      <main className="max-w-6xl mx-auto px-4 py-12">
        {activeTab === "browse" ? <Browse /> : <MySchedule />}
      </main>
    </div>
  );
};

const BackgroundBlobs = () => (
    <div className="fixed inset-0 z-[-1] overflow-hidden bg-slate-50">
      <div className="absolute top-0 left-1/4 w-96 h-96 bg-purple-300 rounded-full mix-blend-multiply filter blur-3xl opacity-30 animate-blob"></div>
      <div className="absolute top-0 right-1/4 w-96 h-96 bg-cyan-300 rounded-full mix-blend-multiply filter blur-3xl opacity-30 animate-blob animation-delay-2000"></div>
      <div className="absolute -bottom-8 left-1/3 w-96 h-96 bg-pink-300 rounded-full mix-blend-multiply filter blur-3xl opacity-30 animate-blob animation-delay-4000"></div>
    </div>
  );

export default Dashboard;
