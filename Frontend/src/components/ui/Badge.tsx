import React from "react";

const Badge = ({
  children,
  className = "",
}: {
  children: React.ReactNode;
  className?: string;
}) => {
  return (
    <span
      className={`px-3 py-1 rounded-full text-xs font-bold ${className}`}
    >
      {children}
    </span>
  );
};

export default Badge;
