"use client"

import { Check } from "lucide-react"

interface TaskCardProps {
  id: string
  content: string
  priority: number
  status: "process" | "done" | "ignore"
  dueDate?: string
  projectName?: string
  onToggleComplete?: (id: string) => void
  onEdit?: (id: string) => void
}

const priorityColors: Record<number, { bg: string; text: string; label: string }> = {
  1: { bg: "bg-red-100", text: "text-red-700", label: "P1" },
  2: { bg: "bg-orange-100", text: "text-orange-700", label: "P2" },
  3: { bg: "bg-yellow-100", text: "text-yellow-700", label: "P3" },
  4: { bg: "bg-slate-100", text: "text-slate-500", label: "P4" },
}

export function TaskCard({
  id,
  content,
  priority,
  status,
  dueDate,
  projectName,
  onToggleComplete,
  onEdit,
}: TaskCardProps) {
  const isCompleted = status === "done"
  const isIgnored = status === "ignore"
  const priorityStyle = priorityColors[priority] || priorityColors[4]

  // Format due date
  const formatDueDate = (date: string) => {
    try {
      const d = new Date(date)
      const now = new Date()
      const isOverdue = d < now && !isCompleted
      const formatted = d.toLocaleDateString("en-US", { month: "short", day: "numeric" })
      return { formatted, isOverdue }
    } catch {
      return { formatted: date, isOverdue: false }
    }
  }

  const dueDateInfo = dueDate ? formatDueDate(dueDate) : null

  return (
    <div
      className={`p-4 bg-white rounded-2xl mb-3 transition-all border-2 ${
        isCompleted ? "opacity-60 border-transparent" : "border-transparent hover:border-slate-200"
      }`}
      onClick={() => onEdit?.(id)}
      role="button"
      tabIndex={0}
    >
      <div className="flex items-start gap-3">
        {/* Checkbox */}
        <button
          onClick={(e) => {
            e.stopPropagation()
            onToggleComplete?.(id)
          }}
          className={`mt-0.5 flex-shrink-0 w-6 h-6 rounded-full border-2 transition-all flex items-center justify-center ${
            isCompleted
              ? "bg-green-500 border-green-500 text-white"
              : isIgnored
              ? "bg-slate-300 border-slate-300 text-white"
              : "border-slate-300 hover:border-slate-400"
          }`}
          aria-label={`Mark "${content}" as ${isCompleted ? "incomplete" : "complete"}`}
        >
          {(isCompleted || isIgnored) && <Check className="w-4 h-4" />}
        </button>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <h3
            className={`font-medium text-sm mb-1.5 break-words ${
              isCompleted ? "line-through text-gray-400" : "text-gray-800"
            }`}
          >
            {content}
          </h3>

          {/* Metadata row */}
          <div className="flex flex-wrap items-center gap-2">
            {/* Priority badge */}
            {priority < 4 && (
              <span
                className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${priorityStyle.bg} ${priorityStyle.text}`}
              >
                {priorityStyle.label}
              </span>
            )}

            {/* Due date */}
            {dueDateInfo && (
              <span
                className={`inline-flex items-center text-xs ${
                  dueDateInfo.isOverdue ? "text-red-500 font-medium" : "text-gray-400"
                }`}
              >
                {dueDateInfo.isOverdue && "⚠ "}
                {dueDateInfo.formatted}
              </span>
            )}

            {/* Project name */}
            {projectName && (
              <span className="inline-flex items-center text-xs text-gray-400">
                📁 {projectName}
              </span>
            )}

            {/* Short ID */}
            <span className="text-xs text-gray-300 ml-auto">{id.slice(0, 8)}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
