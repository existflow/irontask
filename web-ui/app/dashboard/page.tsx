"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { isAuthenticated, logout } from "@/lib/api"
import {
  useAppStore,
  useFilteredTasks,
  useActiveTasks,
  useCompletedTasks,
  useCurrentProject,
  useProjectTaskCount,
} from "@/lib/store"
import { toast } from "sonner"
import {
  Loader2,
  Plus,
  RefreshCw,
  Check,
  Inbox,
  FolderOpen,
  LogOut,
  Archive,
  Pencil,
  CheckCircle2,
  Circle,
} from "lucide-react"
import Link from "next/link"

// Task Card Component
function TaskCard({ task }: { task: { id: string; content: string; status: string; priority: number } }) {
  const { toggleTaskDone, archiveTask, setEditingTask, editingTaskId, editTaskContent, updateTask } = useAppStore()
  const isEditing = editingTaskId === task.id
  const isDone = task.status === "done"

  if (isEditing) {
    return (
      <div className="group bg-white rounded-xl border-2 border-blue-500 p-4 shadow-lg">
        <input
          type="text"
          value={editTaskContent}
          onChange={(e) => useAppStore.setState({ editTaskContent: e.target.value })}
          onBlur={() => {
            if (editTaskContent.trim()) {
              updateTask(task.id, { content: editTaskContent.trim() })
            } else {
              setEditingTask(null)
            }
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter" && editTaskContent.trim()) {
              updateTask(task.id, { content: editTaskContent.trim() })
            } else if (e.key === "Escape") {
              setEditingTask(null)
            }
          }}
          autoFocus
          className="w-full text-gray-900 bg-transparent focus:outline-none"
        />
      </div>
    )
  }

  return (
    <div
      className={`group bg-white rounded-xl border border-gray-100 p-4 shadow-sm hover:shadow-md transition-all duration-200 ${
        isDone ? "opacity-60" : ""
      }`}
    >
      <div className="flex items-start gap-3">
        <button
          onClick={() => toggleTaskDone(task.id)}
          className={`mt-0.5 flex-shrink-0 transition-colors ${
            isDone ? "text-green-500" : "text-gray-300 hover:text-green-500"
          }`}
        >
          {isDone ? (
            <CheckCircle2 className="w-5 h-5" />
          ) : (
            <Circle className="w-5 h-5" />
          )}
        </button>

        <div className="flex-1 min-w-0">
          <p className={`text-gray-900 ${isDone ? "line-through text-gray-500" : ""}`}>
            {task.content}
          </p>
          {task.priority < 4 && (
            <span
              className={`inline-block mt-2 text-xs font-medium px-2 py-0.5 rounded-full ${
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

        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={() => setEditingTask(task.id, task.content)}
            className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg"
          >
            <Pencil className="w-4 h-4" />
          </button>
          <button
            onClick={() => archiveTask(task.id)}
            className="p-1.5 text-gray-400 hover:text-orange-500 hover:bg-orange-50 rounded-lg"
            title="Archive task"
          >
            <Archive className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  )
}

// Project Item Component
function ProjectItem({ project }: { project: { id: string; name: string } }) {
  const {
    selectedProject,
    setSelectedProject,
    editingProjectId,
    editProjectName,
    setEditingProject,
    updateProject,
    archiveProject,
    tasks,
  } = useAppStore()

  const isSelected = selectedProject === project.id
  const isEditing = editingProjectId === project.id
  const taskCount = tasks.filter((t) => t.project_id === project.id && t.status !== "done").length

  if (isEditing) {
    return (
      <div className="px-3 py-2">
        <input
          type="text"
          value={editProjectName}
          onChange={(e) => useAppStore.setState({ editProjectName: e.target.value })}
          onBlur={() => {
            if (editProjectName.trim()) {
              updateProject(project.id, editProjectName.trim())
            } else {
              setEditingProject(null)
            }
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter" && editProjectName.trim()) {
              updateProject(project.id, editProjectName.trim())
            } else if (e.key === "Escape") {
              setEditingProject(null)
            }
          }}
          autoFocus
          className="w-full px-2 py-1.5 text-sm border-2 border-blue-500 rounded-lg focus:outline-none"
        />
      </div>
    )
  }

  return (
    <div
      className={`group flex items-center gap-2 px-3 py-2.5 rounded-xl cursor-pointer transition-all duration-200 ${
        isSelected
          ? "bg-gradient-to-r from-blue-500 to-blue-600 text-white shadow-md"
          : "text-gray-600 hover:bg-gray-100"
      }`}
      onClick={() => setSelectedProject(project.id)}
    >
      {project.id === "inbox" ? (
        <Inbox className={`w-5 h-5 ${isSelected ? "text-white" : "text-gray-400"}`} />
      ) : (
        <FolderOpen className={`w-5 h-5 ${isSelected ? "text-white" : "text-gray-400"}`} />
      )}
      <span className="flex-1 font-medium truncate">{project.name}</span>
      {taskCount > 0 && (
        <span
          className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
            isSelected ? "bg-white/20 text-white" : "bg-gray-200 text-gray-600"
          }`}
        >
          {taskCount}
        </span>
      )}

      {project.id !== "inbox" && !isSelected && (
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={(e) => {
              e.stopPropagation()
              setEditingProject(project.id, project.name)
            }}
            className="p-1 text-gray-400 hover:text-gray-600 rounded"
          >
            <Pencil className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation()
              archiveProject(project.id)
              toast.success("Project archived")
            }}
            className="p-1 text-gray-400 hover:text-orange-500 rounded"
            title="Archive project"
          >
            <Archive className="w-3.5 h-3.5" />
          </button>
        </div>
      )}
    </div>
  )
}

// Main Dashboard
export default function DashboardPage() {
  const router = useRouter()
  const {
    loading,
    syncing,
    projects,
    loadData,
    newTaskContent,
    setNewTaskContent,
    addTask,
    showNewProject,
    setShowNewProject,
    newProjectName,
    setNewProjectName,
    addProject,
  } = useAppStore()

  const activeTasks = useActiveTasks()
  const completedTasks = useCompletedTasks()
  const currentProject = useCurrentProject()

  // Initial load
  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login")
      return
    }
    loadData(true)
  }, [router, loadData])

  const handleRefresh = async () => {
    useAppStore.setState({ syncing: true })
    await loadData(true)
    useAppStore.setState({ syncing: false })
  }

  const handleLogout = async () => {
    await logout()
    router.replace("/login")
  }

  const handleAddTask = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTaskContent.trim()) return
    await addTask(newTaskContent.trim())
  }

  const handleAddProject = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newProjectName.trim()) return
    await addProject(newProjectName.trim())
  }

  if (loading) {
    return (
      <main className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="w-10 h-10 animate-spin text-blue-500 mx-auto" />
          <p className="mt-4 text-gray-500 font-medium">Loading your tasks...</p>
        </div>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
      <div className="flex h-screen">
        {/* Sidebar */}
        <aside className="w-72 bg-white border-r border-gray-200 flex flex-col shadow-sm">
          {/* Header */}
          <div className="p-6 border-b border-gray-100">
            <div className="flex items-center justify-between">
              <h1 className="text-xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                Irontask
              </h1>
              <div className="flex items-center gap-1">
                <button
                  onClick={handleRefresh}
                  disabled={syncing}
                  className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
                >
                  <RefreshCw className={`w-4 h-4 ${syncing ? "animate-spin" : ""}`} />
                </button>
                <button
                  onClick={handleLogout}
                  className="p-2 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>

          {/* Projects */}
          <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
            <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider px-3 mb-3">
              Projects
            </p>
            {projects.map((project) => (
              <ProjectItem key={project.id} project={project} />
            ))}

            {showNewProject ? (
              <form onSubmit={handleAddProject} className="px-3 py-2">
                <input
                  type="text"
                  value={newProjectName}
                  onChange={(e) => setNewProjectName(e.target.value)}
                  onBlur={() => {
                    if (!newProjectName.trim()) setShowNewProject(false)
                  }}
                  onKeyDown={(e) => e.key === "Escape" && setShowNewProject(false)}
                  placeholder="Project name..."
                  autoFocus
                  className="w-full px-3 py-2 text-sm border-2 border-blue-500 rounded-lg focus:outline-none"
                />
              </form>
            ) : (
              <button
                onClick={() => setShowNewProject(true)}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-xl transition-colors"
              >
                <Plus className="w-5 h-5" />
                <span className="font-medium">New Project</span>
              </button>
            )}
          </nav>

          {/* Stats */}
          <div className="p-4 border-t border-gray-100">
            <div className="bg-gradient-to-r from-blue-50 to-purple-50 rounded-xl p-4">
              <p className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                Today's Progress
              </p>
              <div className="flex items-baseline gap-2">
                <span className="text-3xl font-bold text-gray-900">
                  {completedTasks.length}
                </span>
                <span className="text-gray-500">
                  / {activeTasks.length + completedTasks.length} tasks
                </span>
              </div>
            </div>

            {/* Archive Link */}
            <Link
              href="/archive"
              className="flex items-center gap-2 mt-3 px-3 py-2.5 text-gray-500 hover:text-orange-600 hover:bg-orange-50 rounded-xl transition-colors"
            >
              <Archive className="w-5 h-5" />
              <span className="font-medium">Archive</span>
            </Link>
          </div>
        </aside>

        {/* Main Content */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Project Header */}
          <header className="bg-white border-b border-gray-200 px-8 py-6">
            <div className="flex items-center gap-3">
              {currentProject?.id === "inbox" ? (
                <div className="p-2 bg-blue-100 rounded-xl">
                  <Inbox className="w-6 h-6 text-blue-600" />
                </div>
              ) : (
                <div className="p-2 bg-purple-100 rounded-xl">
                  <FolderOpen className="w-6 h-6 text-purple-600" />
                </div>
              )}
              <div>
                <h2 className="text-2xl font-bold text-gray-900">
                  {currentProject?.name || "Select a project"}
                </h2>
                <p className="text-sm text-gray-500">
                  {activeTasks.length} active tasks
                </p>
              </div>
            </div>
          </header>

          {/* Task Input */}
          <div className="px-8 py-4 bg-white border-b border-gray-100">
            <form onSubmit={handleAddTask} className="flex gap-3">
              <div className="flex-1 relative">
                <input
                  type="text"
                  value={newTaskContent}
                  onChange={(e) => setNewTaskContent(e.target.value)}
                  placeholder="Add a new task..."
                  className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                />
              </div>
              <button
                type="submit"
                disabled={!newTaskContent.trim()}
                className="px-6 py-3 bg-gradient-to-r from-blue-500 to-blue-600 text-white font-medium rounded-xl hover:from-blue-600 hover:to-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md hover:shadow-lg"
              >
                <Plus className="w-5 h-5" />
              </button>
            </form>
          </div>

          {/* Tasks List */}
          <div className="flex-1 overflow-y-auto px-8 py-6">
            {activeTasks.length === 0 && completedTasks.length === 0 ? (
              <div className="text-center py-16">
                <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                  <Check className="w-8 h-8 text-gray-400" />
                </div>
                <h3 className="text-lg font-semibold text-gray-700 mb-2">
                  No tasks yet
                </h3>
                <p className="text-gray-500">
                  Add your first task using the input above
                </p>
              </div>
            ) : (
              <div className="space-y-6">
                {/* Active Tasks */}
                {activeTasks.length > 0 && (
                  <div>
                    <h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
                      Active ({activeTasks.length})
                    </h3>
                    <div className="space-y-3">
                      {activeTasks.map((task) => (
                        <TaskCard key={task.id} task={task} />
                      ))}
                    </div>
                  </div>
                )}

                {/* Completed Tasks */}
                {completedTasks.length > 0 && (
                  <div>
                    <h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
                      Completed ({completedTasks.length})
                    </h3>
                    <div className="space-y-3">
                      {completedTasks.map((task) => (
                        <TaskCard key={task.id} task={task} />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  )
}
