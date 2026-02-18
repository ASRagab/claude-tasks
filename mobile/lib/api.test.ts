import { apiClient } from './api';

describe('apiClient', () => {
  beforeEach(() => {
    (apiClient as any).initialized = true;
    apiClient.baseUrl = 'http://localhost:8080';
    apiClient.authToken = 'secret-token';
    (global as any).fetch = jest.fn();
  });

  it('builds request URL and headers for listTasks', async () => {
    (global as any).fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ tasks: [], total: 0 }),
    });

    const result = await apiClient.listTasks();

    expect(result).toEqual({ tasks: [], total: 0 });
    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/tasks',
      expect.objectContaining({
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          Accept: 'application/json',
          Authorization: 'Bearer secret-token',
        }),
      })
    );
  });


  it('requests specific task run by task and run IDs', async () => {
    (global as any).fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ id: 7, task_id: 1 }),
    });

    await apiClient.getTaskRun(1, 7);

    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/tasks/1/runs/7',
      expect.any(Object)
    );
  });

  it('throws when API base URL is not configured', async () => {
    apiClient.baseUrl = '';
    apiClient.authToken = null;
    (apiClient as any).initialized = true;

    await expect(apiClient.listTasks()).rejects.toThrow('API URL not configured');
  });

  it('throws API error when response is not ok', async () => {
    (global as any).fetch.mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => ({ error: 'boom' }),
    });

    await expect(apiClient.listTasks()).rejects.toThrow('boom');
  });


  it('times out hanging requests', async () => {
    jest.useFakeTimers();

    try {
      (global as any).fetch.mockImplementation((_: string, options?: RequestInit) => {
        return new Promise((_, reject) => {
          const signal = options?.signal;
          if (signal) {
            signal.addEventListener('abort', () => {
              const error = new Error('aborted');
              (error as Error & { name: string }).name = 'AbortError';
              reject(error);
            });
          }
        });
      });

      const pendingRequest = apiClient.listTasks();
      jest.advanceTimersByTime(15001);

      await expect(pendingRequest).rejects.toThrow('Request timed out');
    } finally {
      jest.useRealTimers();
    }
  });


  it('surfaces request canceled when upstream signal is aborted', async () => {
    (global as any).fetch.mockImplementation((_: string, options?: RequestInit) => {
      return new Promise((_, reject) => {
        const signal = options?.signal;
        if (signal?.aborted) {
          const error = new Error('aborted');
          (error as Error & { name: string }).name = 'AbortError';
          reject(error);
          return;
        }

        signal?.addEventListener('abort', () => {
          const error = new Error('aborted');
          (error as Error & { name: string }).name = 'AbortError';
          reject(error);
        });
      });
    });

    const controller = new AbortController();
    controller.abort();

    await expect((apiClient as any).request('/tasks', { signal: controller.signal })).rejects.toThrow('Request canceled');
  });
});
