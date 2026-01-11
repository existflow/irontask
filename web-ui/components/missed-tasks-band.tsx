"use client"

import { useState } from "react"
import { ChevronDown } from "lucide-react"

interface MissedTasksBandProps {
  count?: number
}

export function MissedTasksBand({ count = 2 }: MissedTasksBandProps) {
  const [isExpanded, setIsExpanded] = useState(false)

  return (
    <button
      onClick={() => setIsExpanded(!isExpanded)}
      className="w-full bg-slate-500/40 hover:bg-slate-500/50 text-white py-3 px-4 flex items-center justify-between transition-all"
    >
      <span className="font-semibold text-sm">Missed Tasks</span>
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium">{count}</span>
        <ChevronDown size={18} className={`transition-transform ${isExpanded ? "rotate-180" : ""}`} />
      </div>
    </button>
  )
}
