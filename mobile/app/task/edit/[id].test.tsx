import React from 'react';
import { ActivityIndicator } from 'react-native';
import { fireEvent, render } from '@testing-library/react-native';

import EditTaskScreen from './[id]';
import { useTask, useUpdateTask } from '../../../hooks/useTasks';

jest.mock('expo-router', () => ({
  useLocalSearchParams: () => ({ id: '1' }),
  router: {
    back: jest.fn(),
    push: jest.fn(),
  },
}));

jest.mock('@react-native-community/datetimepicker', () => 'DateTimePicker');

jest.mock('../../../hooks/useTasks', () => ({
  useTask: jest.fn(),
  useUpdateTask: jest.fn(),
}));

jest.mock('../../../lib/ThemeContext', () => ({
  useTheme: () => ({
    colors: {
      background: '#000',
      border: '#333',
      surface: '#111',
      surfaceSecondary: '#222',
      textPrimary: '#fff',
      textSecondary: '#ccc',
      textMuted: '#999',
      orange: '#f90',
      error: '#f44',
      inputBackground: '#111',
      cardBackground: '#111',
    },
    shadows: { md: {} },
  }),
}));

jest.mock('../../../lib/ToastContext', () => ({
  useToast: () => ({ showToast: jest.fn() }),
}));

let mutateSpy: jest.Mock;


describe('EditTaskScreen', () => {
  beforeEach(() => {
    mutateSpy = jest.fn();
    (useUpdateTask as jest.Mock).mockReturnValue({ mutate: mutateSpy, isPending: false });
  });

  it('shows loading indicator while task is loading', () => {
    (useTask as jest.Mock).mockReturnValue({ data: undefined, isLoading: true });

    const view = render(<EditTaskScreen />);

    expect(view.UNSAFE_getByType(ActivityIndicator)).toBeTruthy();
  });


  it('shows not found state when task is missing after load', () => {
    (useTask as jest.Mock).mockReturnValue({ data: undefined, isLoading: false });

    const view = render(<EditTaskScreen />);

    expect(view.getByText('Task not found')).toBeTruthy();
  });

  it('renders existing task values once loaded', () => {
    (useTask as jest.Mock).mockReturnValue({
      data: {
        id: 1,
        name: 'Nightly backup',
        prompt: 'backup now',
        cron_expr: '0 0 2 * * *',
        is_one_off: false,
        working_dir: '.',
        model: 'sonnet',
        permission_mode: 'plan',
        enabled: true,
      },
      isLoading: false,
    });

    const view = render(<EditTaskScreen />);

    expect(view.getByDisplayValue('Nightly backup')).toBeTruthy();
    expect(view.getByDisplayValue('backup now')).toBeTruthy();
  });


  it('preserves model and permission_mode in update payload', () => {
    (useTask as jest.Mock).mockReturnValue({
      data: {
        id: 1,
        name: 'Nightly backup',
        prompt: 'backup now',
        cron_expr: '0 0 2 * * *',
        is_one_off: false,
        working_dir: '.',
        model: 'sonnet',
        permission_mode: 'plan',
        enabled: true,
      },
      isLoading: false,
    });

    const view = render(<EditTaskScreen />);
    fireEvent.press(view.getByText('Save Changes'));

    expect(mutateSpy).toHaveBeenCalledWith(
      {
        id: 1,
        task: expect.objectContaining({
          model: 'sonnet',
          permission_mode: 'plan',
        }),
      },
      expect.any(Object)
    );
  });
});
