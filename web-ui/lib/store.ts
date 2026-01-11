import { create } from "zustand"
import { Task, Project, SyncItem, syncPull, syncPush } from "./api"
import {
  getStoredKey,
  hasEncryptionKey,
  decryptTaskContent,
  decryptProjectData,
  encryptTaskContent,
  encryptProjectData,
} from "./crypto"

interface AppState {
  // Data
  tasks: Task[]
  projects: Project[]
  archivedTasks: Task[]
  archivedProjects: Project[]
  syncVersion: number

  // UI State
  selectedProject: string
  loading: boolean
  syncing: boolean

  // Task UI
  newTaskContent: string
  editingTaskId: string | null
  editTaskContent: string

  // Project UI
  showNewProject: boolean
  newProjectName: string
  editingProjectId: string | null
  editProjectName: string

  // Actions
  setSelectedProject: (id: string) => void
  setNewTaskContent: (content: string) => void
  setEditingTask: (id: string | null, content?: string) => void
  setShowNewProject: (show: boolean) => void
  setNewProjectName: (name: string) => void
  setEditingProject: (id: string | null, name?: string) => void

  // Data Actions
  loadData: (fullSync?: boolean) => Promise<boolean>
  addTask: (content: string) => Promise<void>
  updateTask: (taskId: string, updates: Partial<Task>) => Promise<void>
  toggleTaskDone: (taskId: string) => Promise<void>
  archiveTask: (taskId: string) => Promise<void>
  restoreTask: (taskId: string) => Promise<void>
  addProject: (name: string) => Promise<void>
  updateProject: (projectId: string, name: string) => Promise<void>
  archiveProject: (projectId: string) => Promise<boolean>
  restoreProject: (projectId: string) => Promise<void>
}

export const useAppStore = create<AppState>((set, get) => ({
  // Initial state
  tasks: [],
  projects: [],
  archivedTasks: [],
  archivedProjects: [],
  syncVersion: 0,
  selectedProject: "inbox",
  loading: true,
  syncing: false,
  newTaskContent: "",
  editingTaskId: null,
  editTaskContent: "",
  showNewProject: false,
  newProjectName: "",
  editingProjectId: null,
  editProjectName: "",

  // UI Actions
  setSelectedProject: (id) => set({ selectedProject: id }),
  setNewTaskContent: (content) => set({ newTaskContent: content }),
  setEditingTask: (id, content) =>
    set({ editingTaskId: id, editTaskContent: content || "" }),
  setShowNewProject: (show) => set({ showNewProject: show, newProjectName: "" }),
  setNewProjectName: (name) => set({ newProjectName: name }),
  setEditingProject: (id, name) =>
    set({ editingProjectId: id, editProjectName: name || "" }),

  // Load data from server
  loadData: async (fullSync = false) => {
    set({ loading: true })

    if (!hasEncryptionKey()) {
      set({ loading: false })
      return false
    }

    const key = await getStoredKey()
    if (!key) {
      set({ loading: false })
      return false
    }

    try {
      const sinceVersion = fullSync ? 0 : get().syncVersion
      const response = await syncPull(sinceVersion)

      const activeTasks: Task[] = []
      const archivedTasks: Task[] = []
      const activeProjects: Project[] = []
      const archivedProjects: Project[] = []

      for (const item of response.items) {
        try {
          if (item.type === "task" && item.encrypted_content) {
            const content = await decryptTaskContent(key, item.encrypted_content)
            const task: Task = {
              id: item.client_id,
              project_id: item.project_id || "inbox",
              content: content.content,
              status: (item.status as Task["status"]) || "process",
              priority: item.priority || 4,
              due_date: item.due_date,
              sync_version: item.sync_version,
              deleted: item.deleted,
            }
            if (item.deleted) {
              archivedTasks.push(task)
            } else {
              activeTasks.push(task)
            }
          } else if (item.type === "project") {
            let projectName = item.name || "Unnamed"
            let projectSlug = item.slug || item.client_id
            let projectColor: string | undefined

            if (item.encrypted_data) {
              try {
                const data = await decryptProjectData(key, item.encrypted_data)
                projectName = data.name || projectName
                projectSlug = data.slug || projectSlug
                projectColor = data.color
              } catch {
                // Use plaintext fallback
              }
            }

            const project: Project = {
              id: item.client_id,
              slug: projectSlug,
              name: projectName,
              color: projectColor,
              sync_version: item.sync_version,
              deleted: item.deleted,
            }
            if (item.deleted) {
              archivedProjects.push(project)
            } else {
              activeProjects.push(project)
            }
          }
        } catch (err) {
          console.error("Decrypt error:", item.client_id, err)
        }
      }

      if (fullSync) {
        if (!activeProjects.find((p) => p.id === "inbox")) {
          activeProjects.unshift({
            id: "inbox",
            slug: "inbox",
            name: "Inbox",
            sync_version: 0,
            deleted: false,
          })
        }

        set({
          tasks: activeTasks,
          archivedTasks,
          projects: activeProjects,
          archivedProjects,
          syncVersion: response.sync_version,
          loading: false,
        })
      } else {
        const { tasks: prevTasks, projects: prevProjects, archivedTasks: prevArchived, archivedProjects: prevArchivedProjects } = get()

        // Merge active tasks
        const mergedTasks = [...prevTasks]
        for (const task of activeTasks) {
          const idx = mergedTasks.findIndex((t) => t.id === task.id)
          if (idx >= 0) mergedTasks[idx] = task
          else mergedTasks.push(task)
        }
        // Remove any that became archived
        const finalTasks = mergedTasks.filter((t) => !archivedTasks.find((a) => a.id === t.id))

        // Merge archived tasks
        const mergedArchived = [...prevArchived]
        for (const task of archivedTasks) {
          const idx = mergedArchived.findIndex((t) => t.id === task.id)
          if (idx >= 0) mergedArchived[idx] = task
          else mergedArchived.push(task)
        }

        // Merge active projects
        const mergedProjects = [...prevProjects]
        for (const proj of activeProjects) {
          const idx = mergedProjects.findIndex((p) => p.id === proj.id)
          if (idx >= 0) mergedProjects[idx] = proj
          else mergedProjects.push(proj)
        }
        if (!mergedProjects.find((p) => p.id === "inbox")) {
          mergedProjects.unshift({
            id: "inbox",
            slug: "inbox",
            name: "Inbox",
            sync_version: 0,
            deleted: false,
          })
        }
        const finalProjects = mergedProjects.filter((p) => !archivedProjects.find((a) => a.id === p.id))

        // Merge archived projects
        const mergedArchivedProjects = [...prevArchivedProjects]
        for (const proj of archivedProjects) {
          const idx = mergedArchivedProjects.findIndex((p) => p.id === proj.id)
          if (idx >= 0) mergedArchivedProjects[idx] = proj
          else mergedArchivedProjects.push(proj)
        }

        set({
          tasks: finalTasks,
          archivedTasks: mergedArchived,
          projects: finalProjects,
          archivedProjects: mergedArchivedProjects,
          syncVersion: response.sync_version,
          loading: false,
        })
      }

      return true
    } catch (err) {
      console.error("Sync failed:", err)
      set({ loading: false })
      return false
    }
  },

  // Add a new task
  addTask: async (content) => {
    const { selectedProject, tasks } = get()
    const key = await getStoredKey()
    if (!key) return

    const taskId = crypto.randomUUID()
    const newTask: Task = {
      id: taskId,
      project_id: selectedProject,
      content,
      status: "process",
      priority: 4,
      sync_version: 0,
      deleted: false,
    }

    set({ tasks: [...tasks, newTask], newTaskContent: "" })

    try {
      const encryptedContent = await encryptTaskContent(key, content)
      const syncItem: SyncItem = {
        id: taskId,
        client_id: taskId,
        type: "task",
        project_id: selectedProject,
        encrypted_content: encryptedContent,
        status: "process",
        priority: 4,
        sync_version: 0,
        deleted: false,
        client_updated_at: new Date().toISOString(),
      }

      const response = await syncPush([syncItem])
      if (response.updated.length > 0) {
        set({
          tasks: get().tasks.map((t) =>
            t.id === taskId
              ? { ...t, sync_version: response.updated[0].sync_version }
              : t
          ),
        })
      }
    } catch {
      set({ tasks: get().tasks.filter((t) => t.id !== taskId) })
    }
  },

  // Update task
  updateTask: async (taskId, updates) => {
    const { tasks } = get()
    const task = tasks.find((t) => t.id === taskId)
    if (!task) return

    const key = await getStoredKey()
    if (!key) return

    const updatedTask = { ...task, ...updates }

    set({
      tasks: tasks.map((t) => (t.id === taskId ? updatedTask : t)),
      editingTaskId: null,
    })

    try {
      const encryptedContent = await encryptTaskContent(key, updatedTask.content)
      const syncItem: SyncItem = {
        id: taskId,
        client_id: taskId,
        type: "task",
        project_id: updatedTask.project_id,
        encrypted_content: encryptedContent,
        status: updatedTask.status,
        priority: updatedTask.priority,
        due_date: updatedTask.due_date,
        sync_version: task.sync_version,
        deleted: false,
        client_updated_at: new Date().toISOString(),
      }

      const response = await syncPush([syncItem])
      if (response.updated.length > 0) {
        set({
          tasks: get().tasks.map((t) =>
            t.id === taskId
              ? { ...t, sync_version: response.updated[0].sync_version }
              : t
          ),
        })
      }
    } catch {
      set({ tasks })
    }
  },

  // Toggle task done
  toggleTaskDone: async (taskId) => {
    const { tasks } = get()
    const task = tasks.find((t) => t.id === taskId)
    if (!task) return

    const newStatus = task.status === "done" ? "process" : "done"
    await get().updateTask(taskId, { status: newStatus })
  },

  // Archive task (soft delete)
  archiveTask: async (taskId) => {
    const { tasks, archivedTasks } = get()
    const task = tasks.find((t) => t.id === taskId)
    if (!task) return

    const key = await getStoredKey()
    if (!key) return

    const archivedTask = { ...task, deleted: true }

    // Move to archived
    set({
      tasks: tasks.filter((t) => t.id !== taskId),
      archivedTasks: [...archivedTasks, archivedTask],
    })

    try {
      const encryptedContent = await encryptTaskContent(key, task.content)
      const syncItem: SyncItem = {
        id: taskId,
        client_id: taskId,
        type: "task",
        project_id: task.project_id,
        encrypted_content: encryptedContent,
        status: task.status,
        priority: task.priority,
        sync_version: task.sync_version,
        deleted: true,
        client_updated_at: new Date().toISOString(),
      }

      await syncPush([syncItem])
    } catch {
      // Rollback
      set({
        tasks: [...get().tasks, task],
        archivedTasks: get().archivedTasks.filter((t) => t.id !== taskId),
      })
    }
  },

  // Restore task from archive
  restoreTask: async (taskId) => {
    const { tasks, archivedTasks } = get()
    const task = archivedTasks.find((t) => t.id === taskId)
    if (!task) return

    const key = await getStoredKey()
    if (!key) return

    const restoredTask = { ...task, deleted: false }

    // Move back to active
    set({
      archivedTasks: archivedTasks.filter((t) => t.id !== taskId),
      tasks: [...tasks, restoredTask],
    })

    try {
      const encryptedContent = await encryptTaskContent(key, task.content)
      const syncItem: SyncItem = {
        id: taskId,
        client_id: taskId,
        type: "task",
        project_id: task.project_id,
        encrypted_content: encryptedContent,
        status: task.status,
        priority: task.priority,
        sync_version: task.sync_version,
        deleted: false,
        client_updated_at: new Date().toISOString(),
      }

      await syncPush([syncItem])
    } catch {
      // Rollback
      set({
        archivedTasks: [...get().archivedTasks, task],
        tasks: get().tasks.filter((t) => t.id !== taskId),
      })
    }
  },

  // Add project
  addProject: async (name) => {
    const { projects } = get()
    const key = await getStoredKey()
    if (!key) return

    const projectId = name.toLowerCase().replace(/\s+/g, "-")
    const newProject: Project = {
      id: projectId,
      slug: projectId,
      name,
      sync_version: 0,
      deleted: false,
    }

    set({
      projects: [...projects, newProject],
      showNewProject: false,
      newProjectName: "",
      selectedProject: projectId,
    })

    try {
      const encryptedData = await encryptProjectData(key, {
        name,
        slug: projectId,
      })
      const syncItem: SyncItem = {
        id: projectId,
        client_id: projectId,
        type: "project",
        slug: projectId,
        name,
        encrypted_data: encryptedData,
        sync_version: 0,
        deleted: false,
        client_updated_at: new Date().toISOString(),
      }

      const response = await syncPush([syncItem])
      if (response.updated.length > 0) {
        set({
          projects: get().projects.map((p) =>
            p.id === projectId
              ? { ...p, sync_version: response.updated[0].sync_version }
              : p
          ),
        })
      }
    } catch {
      set({
        projects: get().projects.filter((p) => p.id !== projectId),
        selectedProject: "inbox",
      })
    }
  },

  // Update project
  updateProject: async (projectId, name) => {
    const { projects } = get()
    const project = projects.find((p) => p.id === projectId)
    if (!project || projectId === "inbox") return

    const key = await getStoredKey()
    if (!key) return

    set({
      projects: projects.map((p) =>
        p.id === projectId ? { ...p, name } : p
      ),
      editingProjectId: null,
    })

    try {
      const slug = name.toLowerCase().replace(/\s+/g, "-")
      const encryptedData = await encryptProjectData(key, {
        name,
        slug,
        color: project.color,
      })
      const syncItem: SyncItem = {
        id: projectId,
        client_id: projectId,
        type: "project",
        slug,
        name,
        encrypted_data: encryptedData,
        sync_version: project.sync_version,
        deleted: false,
        client_updated_at: new Date().toISOString(),
      }

      const response = await syncPush([syncItem])
      if (response.updated.length > 0) {
        set({
          projects: get().projects.map((p) =>
            p.id === projectId
              ? { ...p, sync_version: response.updated[0].sync_version }
              : p
          ),
        })
      }
    } catch {
      set({ projects })
    }
  },

  // Archive project (soft delete)
  archiveProject: async (projectId) => {
    const { projects, tasks, selectedProject, archivedProjects } = get()
    if (projectId === "inbox") return false

    const project = projects.find((p) => p.id === projectId)
    if (!project) return false

    // Check for active tasks
    const projectTasks = tasks.filter((t) => t.project_id === projectId)
    if (projectTasks.length > 0) {
      return false // Can't archive project with active tasks
    }

    const key = await getStoredKey()
    if (!key) return false

    const archivedProject = { ...project, deleted: true }

    set({
      projects: projects.filter((p) => p.id !== projectId),
      archivedProjects: [...archivedProjects, archivedProject],
      selectedProject: selectedProject === projectId ? "inbox" : selectedProject,
    })

    try {
      const encryptedData = await encryptProjectData(key, {
        name: project.name,
        slug: project.slug,
        color: project.color,
      })
      const syncItem: SyncItem = {
        id: projectId,
        client_id: projectId,
        type: "project",
        slug: project.slug,
        name: project.name,
        encrypted_data: encryptedData,
        sync_version: project.sync_version,
        deleted: true,
        client_updated_at: new Date().toISOString(),
      }

      await syncPush([syncItem])
      return true
    } catch {
      set({
        projects: [...get().projects, project],
        archivedProjects: get().archivedProjects.filter((p) => p.id !== projectId),
      })
      return false
    }
  },

  // Restore project from archive
  restoreProject: async (projectId) => {
    const { projects, archivedProjects } = get()
    const project = archivedProjects.find((p) => p.id === projectId)
    if (!project) return

    const key = await getStoredKey()
    if (!key) return

    const restoredProject = { ...project, deleted: false }

    set({
      archivedProjects: archivedProjects.filter((p) => p.id !== projectId),
      projects: [...projects, restoredProject],
    })

    try {
      const encryptedData = await encryptProjectData(key, {
        name: project.name,
        slug: project.slug,
        color: project.color,
      })
      const syncItem: SyncItem = {
        id: projectId,
        client_id: projectId,
        type: "project",
        slug: project.slug,
        name: project.name,
        encrypted_data: encryptedData,
        sync_version: project.sync_version,
        deleted: false,
        client_updated_at: new Date().toISOString(),
      }

      await syncPush([syncItem])
    } catch {
      set({
        archivedProjects: [...get().archivedProjects, project],
        projects: get().projects.filter((p) => p.id !== projectId),
      })
    }
  },
}))

// Selectors
export const useFilteredTasks = () => {
  const tasks = useAppStore((state) => state.tasks)
  const selectedProject = useAppStore((state) => state.selectedProject)
  return tasks.filter((t) => t.project_id === selectedProject)
}

export const useActiveTasks = () => {
  const filteredTasks = useFilteredTasks()
  return filteredTasks.filter((t) => t.status !== "done")
}

export const useCompletedTasks = () => {
  const filteredTasks = useFilteredTasks()
  return filteredTasks.filter((t) => t.status === "done")
}

export const useCurrentProject = () => {
  const projects = useAppStore((state) => state.projects)
  const selectedProject = useAppStore((state) => state.selectedProject)
  return projects.find((p) => p.id === selectedProject)
}

export const useProjectTaskCount = (projectId: string) => {
  const tasks = useAppStore((state) => state.tasks)
  return tasks.filter((t) => t.project_id === projectId && t.status !== "done").length
}

export const useArchivedTasks = () => {
  return useAppStore((state) => state.archivedTasks)
}

export const useArchivedProjects = () => {
  return useAppStore((state) => state.archivedProjects)
}
