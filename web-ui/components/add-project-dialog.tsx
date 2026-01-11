"use client"

import { useState } from "react"
import { X } from "lucide-react"
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

interface AddProjectDialogProps {
  open: boolean
  onClose: () => void
  onAdd: (name: string, color: string) => void
}

const projectColors = [
  { value: "#4ECDC4", label: "Teal" },
  { value: "#FF6B6B", label: "Red" },
  { value: "#FFB347", label: "Orange" },
  { value: "#FFE66D", label: "Yellow" },
  { value: "#95E1A3", label: "Green" },
  { value: "#7B8794", label: "Gray" },
  { value: "#608d9e", label: "Slate Blue" },
  { value: "#9B59B6", label: "Purple" },
]

export function AddProjectDialog({ open, onClose, onAdd }: AddProjectDialogProps) {
  const [name, setName] = useState("")
  const [color, setColor] = useState("#4ECDC4")
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async () => {
    if (!name.trim()) return

    setIsSubmitting(true)
    try {
      await onAdd(name.trim(), color)
      // Reset form
      setName("")
      setColor("#4ECDC4")
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
          <DrawerTitle>New Project</DrawerTitle>
          <DrawerClose asChild>
            <Button variant="ghost" size="icon-sm">
              <X className="w-4 h-4" />
            </Button>
          </DrawerClose>
        </DrawerHeader>

        <div className="px-4 space-y-4">
          {/* Project Name */}
          <div>
            <label className="text-sm font-medium text-gray-600 mb-2 block">Name</label>
            <Input
              placeholder="Project name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={handleKeyDown}
              autoFocus
              className="text-base"
            />
          </div>

          {/* Color Selection */}
          <div>
            <label className="text-sm font-medium text-gray-600 mb-2 block">Color</label>
            <div className="flex flex-wrap gap-2">
              {projectColors.map((c) => (
                <button
                  key={c.value}
                  onClick={() => setColor(c.value)}
                  className={`w-10 h-10 rounded-full transition-all ${
                    color === c.value ? "ring-2 ring-offset-2 ring-slate-blue scale-110" : ""
                  }`}
                  style={{ backgroundColor: c.value }}
                  title={c.label}
                />
              ))}
            </div>
          </div>

          {/* Preview */}
          <div className="pt-4 border-t">
            <label className="text-sm font-medium text-gray-600 mb-2 block">Preview</label>
            <div
              className="px-4 py-2 rounded-full text-sm font-medium text-white inline-flex items-center"
              style={{ backgroundColor: color }}
            >
              {name || "Project Name"}
            </div>
          </div>
        </div>

        <DrawerFooter>
          <Button
            onClick={handleSubmit}
            disabled={!name.trim() || isSubmitting}
            className="w-full bg-slate-blue hover:bg-slate-blue/90"
          >
            {isSubmitting ? "Creating..." : "Create Project"}
          </Button>
          <Button variant="outline" onClick={onClose} className="w-full">
            Cancel
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  )
}
