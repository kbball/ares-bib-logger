import '@testing-library/jest-dom'
import { server } from './server'

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

// jsdom doesn't implement clipboard — provide a persistent stub so components don't throw
// Tests that need to assert clipboard calls can do Object.assign(navigator.clipboard, { writeText: vi.fn() })
const clipboardStub: Clipboard = {
  writeText: () => Promise.resolve(),
  readText: () => Promise.resolve(''),
  read: () => Promise.resolve(new DataTransfer() as unknown as ClipboardItems),
  write: () => Promise.resolve(),
  addEventListener: () => {},
  removeEventListener: () => {},
  dispatchEvent: () => false,
} as unknown as Clipboard

Object.defineProperty(navigator, 'clipboard', {
  get: () => clipboardStub,
  configurable: true,
})

// This test environment's jsdom doesn't implement window.localStorage — provide
// a real in-memory Storage-compatible stub so components using it work in tests.
// Reset between tests so state doesn't leak across test files.
function createMemoryStorage(): Storage {
  const store = new Map<string, string>()
  return {
    getItem: (key) => store.get(key) ?? null,
    setItem: (key, value) => void store.set(key, String(value)),
    removeItem: (key) => void store.delete(key),
    clear: () => store.clear(),
    key: (index) => Array.from(store.keys())[index] ?? null,
    get length() {
      return store.size
    },
  }
}

Object.defineProperty(window, 'localStorage', {
  value: createMemoryStorage(),
  configurable: true,
  writable: true,
})

afterEach(() => window.localStorage.clear())
