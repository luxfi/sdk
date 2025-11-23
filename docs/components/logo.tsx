import React from "react"

interface LogoProps {
  size?: number
  variant?: "white" | "colored"
  className?: string
}

export const Logo: React.FC<LogoProps> = ({ 
  size = 32, 
  variant = "colored",
  className = "" 
}) => {
  const color = variant === "white" ? "#FFFFFF" : "#0ba5ec"
  
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      <rect width="100" height="100" rx="20" fill={color} fillOpacity="0.1" />
      <path
        d="M30 70V30L50 50L70 30V70"
        stroke={color}
        strokeWidth="6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="50" cy="50" r="8" fill={color} />
    </svg>
  )
}