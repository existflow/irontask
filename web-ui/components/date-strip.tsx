"use client"

import { useState } from "react"

interface DateStripProps {
  activeDay?: number
  onDayChange?: (day: number) => void
}

export function DateStrip({ activeDay = 9, onDayChange }: DateStripProps) {
  const [selected, setSelected] = useState(activeDay)

  const days = Array.from({ length: 7 }, (_, i) => ({
    day: 6 + i,
    name: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"][i],
  }))

  const handleDayClick = (day: number) => {
    setSelected(day)
    onDayChange?.(day)
  }

  return (
    <div className="overflow-x-auto px-4 py-4">
      <div className="flex gap-3 min-w-max">
        {days.map(({ day, name }) => (
          <button
            key={day}
            onClick={() => handleDayClick(day)}
            className={`flex flex-col items-center py-2 px-3 rounded-xl transition-all ${
              selected === day ? "bg-white text-slate-blue font-semibold" : "text-white"
            }`}
          >
            <span className="text-xs font-medium opacity-75">{name}</span>
            <span className="text-lg font-bold">{day}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
