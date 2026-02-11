import React from "react";

const Card = ({
  children,
  className = "",
}: {
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    className={`bg-white/70 backdrop-blur-xl border border-white/20 shadow-xl rounded-2xl ${className}`}
  >
    {children}
  </div>
);

export default Card;
