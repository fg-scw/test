import { useQuery, UseQueryOptions, UseQueryResult } from '@tanstack/react-query'
import { getOpenstackCredentialsList } from 'src/api/openstack-creds/openstackCreds'
import { OpenstackCreds } from 'src/api/openstack-creds/model'

export const OPENSTACK_CREDS_QUERY_KEY = ['openstackCreds']

type Options = Omit<UseQueryOptions<OpenstackCreds[]>, 'queryKey' | 'queryFn'>

export const useOpenstackCredentialsQuery = (
  namespace = undefined,
  options: Options = {}
): UseQueryResult<OpenstackCreds[]> => {
  return useQuery<OpenstackCreds[]>({
    queryKey: [...OPENSTACK_CREDS_QUERY_KEY, namespace],
    queryFn: async () => getOpenstackCredentialsList(namespace),
    staleTime: Infinity,
    refetchOnWindowFocus: true,
    ...options // Override with custom options
  })
}
