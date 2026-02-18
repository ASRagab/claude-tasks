import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../lib/api';
import type { TaskRequest, TaskRunsResponse } from '../lib/types';

const ACTIVE_RUN_POLL_INTERVAL_MS = 3000;

export function getTaskRunsRefetchInterval(data?: TaskRunsResponse): number | false {
  const runs = data?.runs ?? [];
  const hasActiveRun = runs.some((run) => run.status === 'pending' || run.status === 'running');
  return hasActiveRun ? ACTIVE_RUN_POLL_INTERVAL_MS : false;
}

export function useTasks() {
  return useQuery({
    queryKey: ['tasks'],
    queryFn: () => apiClient.listTasks(),
    refetchInterval: 10000,
    refetchIntervalInBackground: false,
  });
}

export function useTask(id: number) {
  return useQuery({
    queryKey: ['tasks', id],
    queryFn: () => apiClient.getTask(id),
    enabled: !!id,
  });
}

export function useTaskRuns(id: number, limit = 20) {
  return useQuery({
    queryKey: ['tasks', id, 'runs', limit],
    queryFn: () => apiClient.getTaskRuns(id, limit),
    enabled: !!id,
    refetchInterval: (query) => getTaskRunsRefetchInterval(query.state.data as TaskRunsResponse | undefined),
    refetchIntervalInBackground: false,
  });
}

export function useTaskRun(taskId: number, runId: number) {
  return useQuery({
    queryKey: ['tasks', taskId, 'runs', runId],
    queryFn: () => apiClient.getTaskRun(taskId, runId),
    enabled: !!taskId && !!runId,
  });
}


export function useCreateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (task: TaskRequest) => apiClient.createTask(task),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
  });
}

export function useUpdateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, task }: { id: number; task: TaskRequest }) =>
      apiClient.updateTask(id, task),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['tasks', id] });
    },
  });
}

export function useDeleteTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => apiClient.deleteTask(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
  });
}

export function useToggleTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => apiClient.toggleTask(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['tasks', id] });
    },
  });
}

export function useRunTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => apiClient.runTask(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['tasks', id] });
      queryClient.invalidateQueries({ queryKey: ['tasks', id, 'runs'] });
    },
  });
}
