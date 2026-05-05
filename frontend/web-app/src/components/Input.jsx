import React from 'react'

const Input = ({ label, ...props }) => {
  return (
    <div className="mb-4">
      {label && (
        <label className="block text-text-secondary text-sm font-medium mb-2">
          {label}
        </label>
      )}
      <input
        className="w-full bg-surface border border-border text-text-primary px-3 py-2 rounded-lg focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary focus:ring-opacity-20"
        {...props}
      />
    </div>
  )
}

export default Input
