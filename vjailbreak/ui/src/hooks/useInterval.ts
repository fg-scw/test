import { useEffect, useRef } from 'react'

export const useInterval = (callback, delay, condition) => {
  const savedCallback = useRef<() => void>()

  useEffect(() => {
    savedCallback.current = callback
  }, [callback])

  useEffect(() => {
    let intervalId: NodeJS.Timeout | null = null

    const tick = () => {
      if (savedCallback.current) {
        savedCallback.current()
      }
    }
    if (delay !== null && condition) {
      intervalId = setInterval(tick, delay)
    }
    return () => {
      if (intervalId !== null) {
        clearInterval(intervalId)
      }
    }
  }, [delay, condition])
}
