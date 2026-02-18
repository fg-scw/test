import { useCallback, useReducer } from 'react'

interface MergeAction<T> {
  type: 'merge'
  payload: Partial<T>
}

interface ReplaceAction<T> {
  type: 'replace'
  payload: T
}

type ParamsReducerAction<T> = MergeAction<T> | ReplaceAction<T>

const paramsReducer = <T>(state: T, action: ParamsReducerAction<T>): T => {
  const { type, payload } = action
  switch (type) {
    case 'merge':
      return { ...state, ...payload }
    case 'replace':
      return payload // Here, payload must be T, not Partial<T>
    default:
      return state
  }
}

export interface UseParamsReturnType<T> {
  params: T
  updateParams: (newParams: Partial<T>) => void
  setParams: (newParams: T) => void
  getParamsUpdater: (key: keyof T) => (value: T[keyof T]) => void
}

const useParams = <T extends Record<string, unknown>>(defaultParams: T): UseParamsReturnType<T> => {
  const [params, dispatch] = useReducer<React.Reducer<T, ParamsReducerAction<T>>>(
    paramsReducer,
    defaultParams
  )

  // Merges new partial values into existing params
  const updateParams = useCallback((value: Partial<T>): void => {
    dispatch({ type: 'merge', payload: value })
  }, [])

  // Replaces params entirely with new values
  const setParams = useCallback((value: T): void => {
    dispatch({ type: 'replace', payload: value })
  }, [])

  // Returns a function to update a specific key's value
  const getParamsUpdater = useCallback(
    (key: keyof T) =>
      (value: T[keyof T]): void => {
        dispatch({ type: 'merge', payload: { [key]: value } as Partial<T> })
      },
    []
  )

  return {
    params,
    updateParams,
    getParamsUpdater,
    setParams
  }
}

export default useParams
