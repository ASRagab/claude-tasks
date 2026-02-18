import React from 'react';
import { Alert } from 'react-native';
import { fireEvent, render } from '@testing-library/react-native';

import NewTaskScreen from '../../../app/task/new';
import { useCreateTask } from '../../../hooks/useTasks';

const mutateSpy = jest.fn();

jest.mock('expo-router', () => ({
  router: {
    back: jest.fn(),
    push: jest.fn(),
  },
}));

jest.mock('@react-native-community/datetimepicker', () => 'DateTimePicker');

jest.mock('../../../hooks/useTasks', () => ({
  useCreateTask: jest.fn(),
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
    },
    shadows: { md: {} },
  }),
}));

jest.mock('../../../lib/ToastContext', () => ({
  useToast: () => ({ showToast: jest.fn() }),
}));

describe('NewTaskScreen', () => {
  let alertSpy: jest.SpyInstance;

  beforeEach(() => {
    mutateSpy.mockReset();
    alertSpy = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    (useCreateTask as jest.Mock).mockReturnValue({ mutate: mutateSpy, isPending: false });
  });

  afterEach(() => {
    alertSpy.mockRestore();
  });

  it('includes model, permission mode, and webhooks in create payload', () => {
    const view = render(<NewTaskScreen />);

    fireEvent.changeText(view.getByPlaceholderText('Task name'), 'Nightly backup');
    fireEvent.changeText(view.getByPlaceholderText('What should Claude do?'), 'Run backup');

    fireEvent.press(view.getByText('One-off'));
    fireEvent.press(view.getByText('Sonnet'));
    fireEvent.press(view.getByText('Plan'));

    fireEvent.changeText(view.getByPlaceholderText('https://discord.com/api/webhooks/...'), ' https://discord.example/webhook ');
    fireEvent.changeText(view.getByPlaceholderText('https://hooks.slack.com/services/...'), ' https://slack.example/webhook ');

    fireEvent.press(view.getByText('Create Task'));

    expect(mutateSpy).toHaveBeenCalledWith(
      {
        name: 'Nightly backup',
        prompt: 'Run backup',
        cron_expr: '',
        working_dir: '.',
        model: 'sonnet',
        permission_mode: 'plan',
        discord_webhook: 'https://discord.example/webhook',
        slack_webhook: 'https://slack.example/webhook',
        enabled: true,
      },
      expect.any(Object)
    );
  });

  it('uses defaults for model and permission mode when unchanged', () => {
    const view = render(<NewTaskScreen />);

    fireEvent.changeText(view.getByPlaceholderText('Task name'), 'Nightly backup');
    fireEvent.changeText(view.getByPlaceholderText('What should Claude do?'), 'Run backup');
    fireEvent.press(view.getByText('One-off'));
    fireEvent.press(view.getByText('Create Task'));

    expect(mutateSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        model: '',
        permission_mode: 'bypassPermissions',
        discord_webhook: undefined,
        slack_webhook: undefined,
      }),
      expect.any(Object)
    );
  });

  it('allows saving task with existing http webhook URL', () => {
    const view = render(<NewTaskScreen />);

    fireEvent.changeText(view.getByPlaceholderText('Task name'), 'Nightly backup');
    fireEvent.changeText(view.getByPlaceholderText('What should Claude do?'), 'Run backup');
    fireEvent.press(view.getByText('One-off'));
    fireEvent.changeText(view.getByPlaceholderText('https://discord.com/api/webhooks/...'), 'http://discord.example/webhook');

    fireEvent.press(view.getByText('Create Task'));

    expect(mutateSpy).toHaveBeenCalled();
    expect(alertSpy).not.toHaveBeenCalled();
  });
});
