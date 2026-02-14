import React from "react";

const Input = ({
  className = "",
  ref,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & {
  className?: string;
  ref?: React.Ref<HTMLInputElement>;
}) => {
  return (
    <div className="relative w-full">
      <input
        {...props}
        ref={ref}
        className={`w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-transparent outline-none transition-all placeholder:text-slate-400 disabled:opacity-50 disabled:cursor-not-allowed ${className}`}
      />
    </div>
  );
};

export default Input;
