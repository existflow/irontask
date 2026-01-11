"use client"

import { Plus } from "lucide-react"

interface FloatingActionButtonProps {
  onClick?: () => void
}

export function FloatingActionButton({ onClick }: FloatingActionButtonProps) {
  return (
    <button
      onClick={onClick}
      className="fixed bottom-20 right-6 w-14 h-14 bg-orange-accent rounded-full flex items-center justify-center text-white shadow-lg hover:shadow-xl hover:scale-110 transition-all z-40"
      aria-label="Add new task"
    >
      <Plus size={24} />
    </button>
  )
}
