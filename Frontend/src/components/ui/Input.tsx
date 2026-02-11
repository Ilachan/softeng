import React from "react";

const Input = ({
  type = "text",
  placeholder,
  className = "",
}: {
  type?: string;
  placeholder?: string;
  className?: string;
}) => {
  return (
    <input
      type={type}
      placeholder={placeholder}
      className={`w-full pl-10 pr-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-transparent outline-none transition-all ${className}`}
    />
  );
};

export default Input;
