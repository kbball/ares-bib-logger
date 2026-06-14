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
