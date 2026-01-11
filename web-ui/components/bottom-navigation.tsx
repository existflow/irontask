"use client"

import { useState } from "react"
import { Calendar, PieChart, Users } from "lucide-react"

interface BottomNavigationProps {
  activeTab?: "calendar" | "analytics" | "profile"
  onTabChange?: (tab: "calendar" | "analytics" | "profile") => void
}

export function BottomNavigation({ activeTab = "calendar", onTabChange }: BottomNavigationProps) {
  const [selected, setSelected] = useState(activeTab)

  const handleTabClick = (tab: "calendar" | "analytics" | "profile") => {
    setSelected(tab)
    onTabChange?.(tab)
  }

  const tabs = [
    { id: "calendar", icon: Calendar, label: "Calendar" },
    { id: "analytics", icon: PieChart, label: "Analytics" },
    { id: "profile", icon: Users, label: "Profile" },
  ] as const

  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-white border-t border-slate-200 px-4 py-2">
      <div className="flex justify-around items-center max-w-md mx-auto">
        {tabs.map(({ id, icon: Icon, label }) => (
          <button
            key={id}
            onClick={() => handleTabClick(id)}
            className={`flex flex-col items-center py-3 px-4 transition-all ${
              selected === id ? "text-orange-accent" : "text-gray-400 hover:text-gray-600"
            }`}
            aria-label={label}
          >
            <Icon size={24} />
            <span className="text-xs mt-1">{label}</span>
          </button>
        ))}
      </div>
    </nav>
  )
}
