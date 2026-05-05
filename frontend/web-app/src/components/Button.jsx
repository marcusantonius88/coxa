import React from 'react'

const Button = ({ children, variant = 'primary', ...props }) => {
  const baseStyles = 'px-4 py-2 rounded-lg font-semibold transition-all duration-300'
  const variants = {
    primary: 'bg-primary text-black hover:bg-primary-hover',
    secondary: 'bg-surface text-text-primary border border-border hover:bg-opacity-80',
  }

  return (
    <button className={`${baseStyles} ${variants[variant]}`} {...props}>
      {children}
    </button>
  )
}

export default Button
