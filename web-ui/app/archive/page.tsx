"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { isAuthenticated } from "@/lib/api"
import {
  useAppStore,
  useArchivedTasks,
  useArchivedProjects,
} from "@/lib/store"
import { toast } from "sonner"
import {
  Loader2,
  ArrowLeft,
  RotateCcw,
  Inbox,
  FolderOpen,
  CheckCircle2,
  Circle,
} from "lucide-react"
import Link from "next/link"

// Archived Task Card
function ArchivedTaskCard({ task }: { task: { id: string; content: string; status: string; priority: number; project_id: string } }) {
  const { restoreTask, projects, archivedProjects } = useAppStore()
  const isDone = task.status === "done"

  // Find project name
  const allProjects = [...projects, ...archivedProjects]
  const project = allProjects.find((p) => p.id === task.project_id)
  const projectName = project?.name || "Unknown"

  const handleRestore = async () => {
    await restoreTask(task.id)
    toast.success("Task restored")
  }

  return (
    <div className="group bg-white rounded-xl border border-gray-100 p-4 shadow-sm hover:shadow-md transition-all duration-200 opacity-70 hover:opacity-100">
      <div className="flex items-start gap-3">
        <div className="mt-0.5 flex-shrink-0 text-gray-300">
          {isDone ? (
            <CheckCircle2 className="w-5 h-5" />
          ) : (
            <Circle className="w-5 h-5" />
          )}
        </div>

        <div className="flex-1 min-w-0">
          <p className="text-gray-600 line-through">{task.content}</p>
          <div className="flex items-center gap-2 mt-2">
            <span className="text-xs text-gray-400">{projectName}</span>
            {task.priority < 4 && (
              <span
                className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                  task.priority === 1
                    ? "bg-red-100 text-red-700"
                    : task.priority === 2
                    ? "bg-orange-100 text-orange-700"
                    : "bg-yellow-100 text-yellow-700"
                }`}
              >
                P{task.priority}
              </span>
            )}
          </div>
        </div>

        <button
          onClick={handleRestore}
          className="p-2 text-gray-400 hover:text-green-500 hover:bg-green-50 rounded-lg transition-colors"
          title="Restore task"
        >
          <RotateCcw className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}

// Archived Project Card
function ArchivedProjectCard({ project }: { project: { id: string; name: string } }) {
  const { restoreProject } = useAppStore()

  const handleRestore = async () => {
    await restoreProject(project.id)
    toast.success("Project restored")
  }

  return (
    <div className="group flex items-center gap-3 bg-white rounded-xl border border-gray-100 p-4 shadow-sm hover:shadow-md transition-all duration-200 opacity-70 hover:opacity-100">
      <FolderOpen className="w-5 h-5 text-gray-400" />
      <span className="flex-1 font-medium text-gray-600 line-through">{project.name}</span>
      <button
        onClick={handleRestore}
        className="p-2 text-gray-400 hover:text-green-500 hover:bg-green-50 rounded-lg transition-colors"
        title="Restore project"
      >
        <RotateCcw className="w-4 h-4" />
      </button>
    </div>
  )
}

export default function ArchivePage() {
  const router = useRouter()
  const { loading, loadData } = useAppStore()
  const archivedTasks = useArchivedTasks()
  const archivedProjects = useArchivedProjects()

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login")
      return
    }
    loadData(true)
  }, [router, loadData])

  if (loading) {
    return (
      <main className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="w-10 h-10 animate-spin text-blue-500 mx-auto" />
          <p className="mt-4 text-gray-500 font-medium">Loading archive...</p>
        </div>
      </main>
    )
  }

  const isEmpty = archivedTasks.length === 0 && archivedProjects.length === 0

  return (
    <main className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
      <div className="max-w-4xl mx-auto px-6 py-8">
        {/* Header */}
        <div className="flex items-center gap-4 mb-8">
          <Link
            href="/dashboard"
            className="p-2 text-gray-500 hover:text-gray-700 hover:bg-white rounded-lg transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Archive</h1>
            <p className="text-sm text-gray-500">
              {archivedTasks.length} tasks, {archivedProjects.length} projects archived
            </p>
          </div>
        </div>

        {isEmpty ? (
          <div className="text-center py-16 bg-white rounded-2xl shadow-sm">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <Inbox className="w-8 h-8 text-gray-400" />
            </div>
            <h3 className="text-lg font-semibold text-gray-700 mb-2">
              Archive is empty
            </h3>
            <p className="text-gray-500">
              Archived tasks and projects will appear here
            </p>
          </div>
        ) : (
          <div className="space-y-8">
            {/* Archived Projects */}
            {archivedProjects.length > 0 && (
              <div>
                <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-4">
                  Archived Projects ({archivedProjects.length})
                </h2>
                <div className="space-y-3">
                  {archivedProjects.map((project) => (
                    <ArchivedProjectCard key={project.id} project={project} />
                  ))}
                </div>
              </div>
            )}

            {/* Archived Tasks */}
            {archivedTasks.length > 0 && (
              <div>
                <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-4">
                  Archived Tasks ({archivedTasks.length})
                </h2>
                <div className="space-y-3">
                  {archivedTasks.map((task) => (
                    <ArchivedTaskCard key={task.id} task={task} />
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </main>
  )
}
