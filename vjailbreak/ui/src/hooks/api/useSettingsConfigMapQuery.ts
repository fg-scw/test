import { useQuery, UseQueryResult } from '@tanstack/react-query'
import { getSettingsConfigMap, VERSION_NAMESPACE, VjailbreakSettings } from 'src/api/settings'

export const useSettingsConfigMapQuery = (
  namespace: string = VERSION_NAMESPACE
): UseQueryResult<VjailbreakSettings> => {
  return useQuery<VjailbreakSettings>({
    queryKey: ['settingsConfigMap', namespace],
    queryFn: async () => getSettingsConfigMap(namespace),
    staleTime: Infinity,
    refetchOnWindowFocus: true
  })
}
