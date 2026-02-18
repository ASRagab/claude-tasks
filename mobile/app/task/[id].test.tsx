import React from 'react';
import { fireEvent, render } from '@testing-library/react-native';

import TaskDetailScreen from './[id]';
import {
  useTask,
  useTaskRuns,
  useToggleTask,
  useRunTask,
  useDeleteTask,
} from '../../hooks/useTasks';

const runMutate = jest.fn();

jest.mock('expo-router', () => {
  const mockRouter = {
    back: jest.fn(),
    push: jest.fn(),
  };

  return {
    useLocalSearchParams: () => ({ id: '1' }),
    router: mockRouter,
    Link: ({ children }: any) => children,
  };
});

jest.mock('expo-glass-effect', () => ({
  GlassView: ({ children }: any) => children,
  isLiquidGlassAvailable: () => false,
}));

jest.mock('../../hooks/useTasks', () => ({
  useTask: jest.fn(),
  useTaskRuns: jest.fn(),
  useToggleTask: jest.fn(),
  useRunTask: jest.fn(),
  useDeleteTask: jest.fn(),
}));

jest.mock('../../lib/ThemeContext', () => ({
  useTheme: () => ({
    colors: {
      background: '#000',
      border: '#333',
      cardBackground: '#111',
      textPrimary: '#fff',
      textSecondary: '#ccc',
      textMuted: '#999',
      orange: '#f90',
      success: '#0f0',
      error: '#f44',
      surfaceSecondary: '#222',
    },
    shadows: { md: {} },
  }),
}));

jest.mock('../../lib/ToastContext', () => ({
  useToast: () => ({ showToast: jest.fn() }),
}));

describe('TaskDetailScreen', () => {
  beforeEach(() => {
    runMutate.mockReset();
    const { router } = jest.requireMock('expo-router');
    router.push.mockReset();

    (useTask as jest.Mock).mockReturnValue({
      data: {
        id: 1,
        name: 'Nightly backup',
        prompt: 'backup now',
        cron_expr: '0 0 2 * * *',
        is_one_off: false,
        working_dir: '.',
        enabled: true,
      },
      isLoading: false,
      refetch: jest.fn(),
    });

    (useTaskRuns as jest.Mock).mockReturnValue({
      data: { runs: [], total: 0 },
      isLoading: false,
      refetch: jest.fn(),
    });

    (useToggleTask as jest.Mock).mockReturnValue({ mutate: jest.fn(), isPending: false });
    (useRunTask as jest.Mock).mockReturnValue({ mutate: runMutate, isPending: false });
    (useDeleteTask as jest.Mock).mockReturnValue({ mutate: jest.fn(), isPending: false });
  });

  it('renders task details', () => {
    const view = render(<TaskDetailScreen />);
    expect(view.getByText('Nightly backup')).toBeTruthy();
    expect(view.getByText('backup now')).toBeTruthy();
  });

  it('invokes run mutation when Run is pressed', () => {
    const view = render(<TaskDetailScreen />);
    fireEvent.press(view.getByText('Run'));

    expect(runMutate).toHaveBeenCalledTimes(1);
    expect(runMutate).toHaveBeenCalledWith(1, expect.any(Object));
  });


  it('does not invoke run mutation while run request is pending', () => {
    (useRunTask as jest.Mock).mockReturnValue({ mutate: runMutate, isPending: true });

    const view = render(<TaskDetailScreen />);
    fireEvent.press(view.getByText('Run'));

    expect(runMutate).not.toHaveBeenCalled();
  });


  it('navigates to run output with identifiers only', () => {
    (useTaskRuns as jest.Mock).mockReturnValue({
      data: {
        runs: [
          {
            id: 77,
            task_id: 1,
            status: 'completed',
            output: 'x'.repeat(5000),
            error: '',
            started_at: '2026-02-17T00:00:00Z',
            ended_at: '2026-02-17T00:00:05Z',
            duration_ms: 5000,
          },
        ],
        total: 1,
      },
      isLoading: false,
      refetch: jest.fn(),
    });

    const view = render(<TaskDetailScreen />);
    fireEvent.press(view.getByText('View Output →'));

    const { router } = jest.requireMock('expo-router');

    expect(router.push).toHaveBeenCalledWith({
      pathname: '/run/[id]',
      params: {
        id: '77',
        taskId: '1',
      },
    });
  });
});
