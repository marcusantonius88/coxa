import React from 'react'

const Card = ({ children, title }) => {
  return (
    <div className="bg-surface border border-border rounded-xl p-6 shadow-lg">
      {title && <h3 className="text-xl font-bold text-text-primary mb-4">{title}</h3>}
      {children}
    </div>
  )
}

export default Card
