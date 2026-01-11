"use client"

import { useState, useEffect } from "react"
import { X, Calendar, Trash2 } from "lucide-react"
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
import type { Task } from "@/lib/api"

interface EditTaskDialogProps {
  open: boolean
  task: Task | null
  onClose: () => void
  onSave: (task: Task) => void
  onDelete: (taskId: string) => void
}

const priorities = [
  { value: 1, label: "P1", color: "bg-red-500" },
  { value: 2, label: "P2", color: "bg-orange-500" },
  { value: 3, label: "P3", color: "bg-yellow-500" },
  { value: 4, label: "P4", color: "bg-slate-400" },
]

export function EditTaskDialog({ open, task, onClose, onSave, onDelete }: EditTaskDialogProps) {
  const [content, setContent] = useState("")
  const [priority, setPriority] = useState(4)
  const [dueDate, setDueDate] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  // Reset form when task changes
  useEffect(() => {
    if (task) {
      setContent(task.content)
      setPriority(task.priority)
      setDueDate(task.due_date || "")
      setShowDeleteConfirm(false)
    }
  }, [task])

  const handleSave = async () => {
    if (!task || !content.trim()) return

    setIsSubmitting(true)
    try {
      await onSave({
        ...task,
        content: content.trim(),
        priority,
        due_date: dueDate || undefined,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!task) return

    setIsSubmitting(true)
    try {
      await onDelete(task.id)
    } finally {
      setIsSubmitting(false)
      setShowDeleteConfirm(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSave()
    }
  }

  if (!task) return null

  return (
    <Drawer open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DrawerContent>
        <DrawerHeader className="flex flex-row items-center justify-between">
          <DrawerTitle>Edit Task</DrawerTitle>
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

          {/* Delete Section */}
          <div className="pt-4 border-t">
            {showDeleteConfirm ? (
              <div className="space-y-2">
                <p className="text-sm text-gray-600">Are you sure you want to delete this task?</p>
                <div className="flex gap-2">
                  <Button
                    variant="destructive"
                    onClick={handleDelete}
                    disabled={isSubmitting}
                    className="flex-1"
                  >
                    {isSubmitting ? "Deleting..." : "Yes, Delete"}
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setShowDeleteConfirm(false)}
                    className="flex-1"
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <button
                onClick={() => setShowDeleteConfirm(true)}
                className="flex items-center gap-2 text-red-500 hover:text-red-600 text-sm font-medium"
              >
                <Trash2 className="w-4 h-4" />
                Delete Task
              </button>
            )}
          </div>
        </div>

        <DrawerFooter>
          <Button
            onClick={handleSave}
            disabled={!content.trim() || isSubmitting}
            className="w-full bg-slate-blue hover:bg-slate-blue/90"
          >
            {isSubmitting ? "Saving..." : "Save Changes"}
          </Button>
          <Button variant="outline" onClick={onClose} className="w-full">
            Cancel
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  )
}
