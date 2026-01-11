"use client"

import { useState } from "react"
import { X, Calendar } from "lucide-react"
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerFooter,
  DrawerClose,
} from "@/components/ui/drawer"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

interface AddTaskDialogProps {
  open: boolean
  onClose: () => void
  onAdd: (content: string, priority: number, dueDate?: string) => void
  projectName?: string
}

const priorities = [
  { value: 1, label: "P1", color: "bg-red-500" },
  { value: 2, label: "P2", color: "bg-orange-500" },
  { value: 3, label: "P3", color: "bg-yellow-500" },
  { value: 4, label: "P4", color: "bg-slate-400" },
]

export function AddTaskDialog({ open, onClose, onAdd, projectName }: AddTaskDialogProps) {
  const [content, setContent] = useState("")
  const [priority, setPriority] = useState(4)
  const [dueDate, setDueDate] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async () => {
    if (!content.trim()) return

    setIsSubmitting(true)
    try {
      await onAdd(content.trim(), priority, dueDate || undefined)
      // Reset form
      setContent("")
      setPriority(4)
      setDueDate("")
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <Drawer open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DrawerContent>
        <DrawerHeader className="flex flex-row items-center justify-between">
          <DrawerTitle>Add Task {projectName && `to ${projectName}`}</DrawerTitle>
          <DrawerClose asChild>
            <Button variant="ghost" size="icon-sm">
              <X className="w-4 h-4" />
            </Button>
          </DrawerClose>
        </DrawerHeader>

        <div className="px-4 space-y-4">
          {/* Task Content */}
          <div>
            <Input
              placeholder="What needs to be done?"
              value={content}
              onChange={(e) => setContent(e.target.value)}
              onKeyDown={handleKeyDown}
              autoFocus
              className="text-base"
            />
          </div>

          {/* Priority Selection */}
          <div>
            <label className="text-sm font-medium text-gray-600 mb-2 block">Priority</label>
            <div className="flex gap-2">
              {priorities.map((p) => (
                <button
                  key={p.value}
                  onClick={() => setPriority(p.value)}
                  className={`px-4 py-2 rounded-full text-sm font-medium transition-all ${
                    priority === p.value
                      ? `${p.color} text-white`
                      : "bg-gray-100 text-gray-600 hover:bg-gray-200"
                  }`}
                >
                  {p.label}
                </button>
              ))}
            </div>
          </div>

          {/* Due Date */}
          <div>
            <label className="text-sm font-medium text-gray-600 mb-2 block">Due Date</label>
            <div className="relative">
              <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
        </div>

        <DrawerFooter>
          <Button
            onClick={handleSubmit}
            disabled={!content.trim() || isSubmitting}
            className="w-full bg-slate-blue hover:bg-slate-blue/90"
          >
            {isSubmitting ? "Adding..." : "Add Task"}
          </Button>
          <Button variant="outline" onClick={onClose} className="w-full">
            Cancel
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  )
}
