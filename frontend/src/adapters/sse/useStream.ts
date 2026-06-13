import { useEffect } from 'react'
import type { SSEEvent } from '../../domain/types'

type Handler<T> = (payload: T) => void

export interface StreamHandlers {
  onBibLogged?: Handler<SSEEvent<unknown>['payload']>
  onSessionChanged?: Handler<SSEEvent<unknown>['payload']>
}

export function useStream(handlers: StreamHandlers) {
  useEffect(() => {
    const es = new EventSource('/api/stream')

    es.onmessage = (e: MessageEvent) => {
      const event = JSON.parse(e.data) as SSEEvent
      if (event.type === 'bib_logged') handlers.onBibLogged?.(event.payload)
      if (event.type === 'session_changed') handlers.onSessionChanged?.(event.payload)
    }

    es.onerror = () => {
      // Browser auto-reconnects on error; nothing to do here.
    }

    return () => es.close()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
}
