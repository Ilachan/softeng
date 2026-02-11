import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Input from "../components/ui/Input";
import { Icons } from "../lib/icons";
import { useAuthStore } from "../store/authStore";

const Login = () => {
  const navigate = useNavigate();
  const { login } = useAuthStore();

  const handleLogin = () => {
    login();
    navigate("/dashboard");
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative">
      <BackgroundBlobs />
      <Card className="w-full max-w-md p-8 relative z-10">
        <div className="text-center mb-8">
          <div className="w-16 h-16 bg-gradient-to-tr from-violet-600 to-indigo-600 rounded-2xl mx-auto flex items-center justify-center mb-4 shadow-lg shadow-indigo-500/30">
            <span className="text-3xl">💪</span>
          </div>
          <h1 className="text-3xl font-bold text-slate-800 mb-2">FitFlow</h1>
          <p className="text-slate-500">Let's get moving.</p>
        </div>

        <div className="space-y-4">
          <div className="relative group">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-slate-400 group-focus-within:text-indigo-500 transition-colors">
              <Icons.Mail />
            </div>
            <Input type="email" placeholder="Email Address" />
          </div>

          <div className="relative group">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-slate-400 group-focus-within:text-indigo-500 transition-colors">
              <Icons.Lock />
            </div>
            <Input type="password" placeholder="Password" />
          </div>

          <Button className="w-full py-3.5 text-lg" onClick={handleLogin}>
            Sign In
          </Button>
        </div>

        <div className="mt-8 text-center text-sm text-slate-500">
          <button
            onClick={() => navigate("/register")}
            className="font-semibold text-indigo-600 hover:text-indigo-700 hover:underline"
          >
            Don't have an account? Sign up
          </button>
        </div>
      </Card>
    </div>
  );
};

import BackgroundBlobs from "../components/ui/BackgroundBlobs";

export default Login;