import { useQuery, useMutation, useQueryClient, UseQueryOptions, UseQueryResult } from '@tanstack/react-query'
import {
  getArrayCredentialsList,
  getArrayCredentials,
  deleteArrayCredentials
} from 'src/api/array-creds/arrayCreds'
import { ArrayCreds } from 'src/api/array-creds/model'

export const ARRAY_CREDS_QUERY_KEY = ['arrayCreds']

type Options = Omit<UseQueryOptions<ArrayCreds[]>, 'queryKey' | 'queryFn'>

export const useArrayCredentialsQuery = (
  namespace?: string,
  options: Options = {}
): UseQueryResult<ArrayCreds[]> => {
  return useQuery<ArrayCreds[]>({
    queryKey: [...ARRAY_CREDS_QUERY_KEY, namespace],
    queryFn: async () => getArrayCredentialsList(namespace),
    staleTime: Infinity,
    refetchOnWindowFocus: true,
    ...options
  })
}

export const useArrayCredentialsByIdQuery = (
  name: string,
  namespace?: string,
  options: Omit<UseQueryOptions<ArrayCreds>, 'queryKey' | 'queryFn'> = {}
) => {
  return useQuery<ArrayCreds>({
    queryKey: [...ARRAY_CREDS_QUERY_KEY, name, namespace],
    queryFn: () => getArrayCredentials(name, namespace),
    enabled: !!name,
    ...options
  })
}

export const useDeleteArrayCredsMutation = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (name: string) => deleteArrayCredentials(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ARRAY_CREDS_QUERY_KEY })
    }
  })
}
