import React from 'react';
import { render } from '@testing-library/react-native';

import RunOutputScreen from './[id]';
import { useTask, useTaskRun } from '../../hooks/useTasks';

const mockUseLocalSearchParams = jest.fn();

jest.mock('expo-router', () => ({
  useLocalSearchParams: () => mockUseLocalSearchParams(),
}));

jest.mock('expo-glass-effect', () => ({
  GlassView: ({ children }: any) => children,
  isLiquidGlassAvailable: () => false,
}));

jest.mock('../../hooks/useTasks', () => ({
  useTask: jest.fn(),
  useTaskRun: jest.fn(),
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

describe('RunOutputScreen', () => {
  beforeEach(() => {
    mockUseLocalSearchParams.mockReturnValue({ id: '77', taskId: '1' });

    (useTask as jest.Mock).mockReturnValue({
      data: {
        id: 1,
        name: 'Nightly backup',
      },
      isLoading: false,
    });

    (useTaskRun as jest.Mock).mockReturnValue({
      data: {
        id: 77,
        task_id: 1,
        status: 'completed',
        output: 'Execution output',
        error: '',
        started_at: '2026-02-17T00:00:00Z',
        ended_at: '2026-02-17T00:00:05Z',
        duration_ms: 5000,
      },
      isLoading: false,
    });
  });

  it('fetches run details from API data using run and task identifiers', () => {
    const view = render(<RunOutputScreen />);

    expect(useTaskRun).toHaveBeenCalledWith(1, 77);
    expect(view.getByText('Nightly backup')).toBeTruthy();
    expect(view.getByText('Execution output')).toBeTruthy();
  });

  it('shows not found state for invalid route params', () => {
    mockUseLocalSearchParams.mockReturnValue({ id: '77' });

    const view = render(<RunOutputScreen />);

    expect(view.getByText('Run not found')).toBeTruthy();
  });
});
