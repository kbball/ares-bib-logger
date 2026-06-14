import { describe, it, expect } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/server'
import { get, post, put, del, ApiError } from './client'

describe('get', () => {
  it('fetches and returns JSON', async () => {
    server.use(http.get('/test', () => HttpResponse.json({ ok: true })))
    const result = await get<{ ok: boolean }>('/test')
    expect(result).toEqual({ ok: true })
  })

  it('throws ApiError on non-OK response with json body', async () => {
    server.use(http.get('/test', () => HttpResponse.json({ error: 'not found' }, { status: 404 })))
    await expect(get('/test')).rejects.toMatchObject({ status: 404, message: 'not found' })
  })

  it('throws ApiError on non-OK response with non-json body', async () => {
    server.use(http.get('/test', () => new HttpResponse('Internal Server Error', { status: 500 })))
    await expect(get('/test')).rejects.toBeInstanceOf(ApiError)
  })
})

describe('post', () => {
  it('sends POST and returns JSON', async () => {
    server.use(http.post('/test', () => HttpResponse.json({ created: true }, { status: 201 })))
    const result = await post<{ created: boolean }>('/test', { name: 'foo' })
    expect(result).toEqual({ created: true })
  })

  it('sends POST with no body', async () => {
    server.use(http.post('/test', () => HttpResponse.json({ ok: true })))
    const result = await post<{ ok: boolean }>('/test')
    expect(result).toEqual({ ok: true })
  })
})

describe('put', () => {
  it('returns undefined on 204', async () => {
    server.use(http.put('/test', () => new HttpResponse(null, { status: 204 })))
    const result = await put<void>('/test')
    expect(result).toBeUndefined()
  })

  it('returns JSON on 200', async () => {
    server.use(http.put('/test', () => HttpResponse.json({ updated: true })))
    const result = await put<{ updated: boolean }>('/test', { x: 1 })
    expect(result).toEqual({ updated: true })
  })
})

describe('del', () => {
  it('returns undefined on 204', async () => {
    server.use(http.delete('/test', () => new HttpResponse(null, { status: 204 })))
    const result = await del<void>('/test')
    expect(result).toBeUndefined()
  })
})

describe('ApiError', () => {
  it('has the correct name and status', () => {
    const err = new ApiError(422, 'invalid')
    expect(err.name).toBe('ApiError')
    expect(err.status).toBe(422)
    expect(err.message).toBe('invalid')
    expect(err).toBeInstanceOf(Error)
  })
})
