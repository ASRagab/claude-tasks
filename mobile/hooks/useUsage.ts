import { useEffect, useState } from 'react';
import { AppState, type AppStateStatus } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../lib/api';

const USAGE_POLL_INTERVAL_MS = 30000;

export function getUsageRefetchInterval(appState: AppStateStatus): number | false {
  return appState === 'active' ? USAGE_POLL_INTERVAL_MS : false;
}

export function useUsage() {
  const [appState, setAppState] = useState<AppStateStatus>(AppState.currentState ?? 'active');

  useEffect(() => {
    const subscription = AppState.addEventListener('change', setAppState);
    return () => {
      subscription.remove();
    };
  }, []);

  return useQuery({
    queryKey: ['usage'],
    queryFn: () => apiClient.getUsage(),
    refetchInterval: getUsageRefetchInterval(appState),
    refetchIntervalInBackground: false,
  });
}
