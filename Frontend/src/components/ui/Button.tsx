import React from "react";

const Button = ({
  children,
  variant = "primary",
  onClick,
  className = "",
}: {
  children: React.ReactNode;
  variant?: "primary" | "secondary" | "danger" | "ghost" | "outline";
  onClick?: () => void;
  className?: string;
}) => {
  const baseStyle =
    "px-5 py-2.5 rounded-xl font-semibold transition-all duration-300 transform active:scale-95 flex items-center justify-center gap-2";
  const variants = {
    primary:
      "bg-gradient-to-r from-violet-600 to-indigo-600 text-white shadow-lg shadow-indigo-500/30 hover:shadow-indigo-500/50 hover:brightness-110",
    secondary:
      "bg-white text-slate-700 border border-slate-200 hover:bg-slate-50 hover:border-slate-300 shadow-sm",
    danger: "bg-red-50 text-red-600 border border-red-100 hover:bg-red-100",
    ghost: "text-slate-500 hover:text-slate-800 hover:bg-slate-100/50",
    outline: "border-2 border-indigo-600 text-indigo-600 hover:bg-indigo-50",
  };
  return (
    <button
      onClick={onClick}
      className={`${baseStyle} ${variants[variant]} ${className}`}
    >
      {children}
    </button>
  );
};

export default Button;
