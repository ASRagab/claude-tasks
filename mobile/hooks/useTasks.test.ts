import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react-native';

import { apiClient } from '../lib/api';
import { getTaskRunsRefetchInterval, useTask, useTaskRun, useTasks } from './useTasks';

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: Infinity,
      },
      mutations: {
        retry: false,
        gcTime: Infinity,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('useTasks hooks', () => {
  beforeEach(() => {
    jest.restoreAllMocks();
  });

  it('loads task list data', async () => {
    const listSpy = jest.spyOn(apiClient, 'listTasks').mockResolvedValue({
      tasks: [
        {
          id: 1,
          name: 'task-1',
          prompt: 'run',
          cron_expr: '0 * * * * *',
          is_one_off: false,
          working_dir: '.',
          enabled: true,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
      ],
      total: 1,
    } as any);

    const { result, unmount } = renderHook(() => useTasks(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(listSpy).toHaveBeenCalledTimes(1);
    expect(result.current.data?.total).toBe(1);

    unmount();
  });

  it('does not fetch task when id is falsy', async () => {
    const getTaskSpy = jest.spyOn(apiClient, 'getTask').mockResolvedValue({} as any);

    const { unmount } = renderHook(() => useTask(0), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(getTaskSpy).not.toHaveBeenCalled();
    });

    unmount();
  });


  it('loads a specific task run by task and run IDs', async () => {
    const getTaskRunSpy = jest.spyOn(apiClient, 'getTaskRun').mockResolvedValue({
      id: 7,
      task_id: 1,
      status: 'completed',
      output: 'done',
      error: '',
      started_at: '2026-01-01T00:00:00Z',
      ended_at: '2026-01-01T00:00:01Z',
      duration_ms: 1000,
    } as any);

    const { result, unmount } = renderHook(() => useTaskRun(1, 7), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(getTaskRunSpy).toHaveBeenCalledWith(1, 7);

    unmount();
  });


  it('polls task runs only when there are active runs', () => {
    expect(getTaskRunsRefetchInterval(undefined)).toBe(false);
    expect(getTaskRunsRefetchInterval({ runs: [], total: 0 })).toBe(false);

    expect(getTaskRunsRefetchInterval({
      runs: [{ status: 'completed' } as any],
      total: 1,
    })).toBe(false);

    expect(getTaskRunsRefetchInterval({
      runs: [{ status: 'running' } as any],
      total: 1,
    })).toBe(3000);

    expect(getTaskRunsRefetchInterval({
      runs: [{ status: 'pending' } as any],
      total: 1,
    })).toBe(3000);
  });
});
