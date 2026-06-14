import { useEffect, useRef } from 'react'
import type { SSEEvent } from '../../domain/types'

type Handler<T> = (payload: T) => void

export interface StreamHandlers {
  onBibLogged?: Handler<SSEEvent<unknown>['payload']>
  onSessionChanged?: Handler<SSEEvent<unknown>['payload']>
}

export function useStream(handlers: StreamHandlers) {
  // Keep a ref that always points to the latest handlers so the EventSource
  // callback is never stale without recreating the connection on every render.
  const handlersRef = useRef(handlers)
  useEffect(() => {
    handlersRef.current = handlers
  })

  useEffect(() => {
    const es = new EventSource('/api/stream')

    es.onmessage = (e: MessageEvent) => {
      const event = JSON.parse(e.data) as SSEEvent
      if (event.type === 'bib_logged') handlersRef.current.onBibLogged?.(event.payload)
      if (event.type === 'session_changed') handlersRef.current.onSessionChanged?.(event.payload)
    }

    es.onerror = () => {
      // Browser auto-reconnects on error; nothing to do here.
    }

    return () => es.close()
  }, [])
}
