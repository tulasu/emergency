export default defineEventHandler(async (event) => {
  const path = getRequestURL(event).pathname
  const accept = getHeader(event, 'accept') ?? ''
  if (accept.includes('text/html')) {
    return
  }

  const prefixes = [
    '/modules',
    '/lessons',
    '/variants',
    '/tickets',
    '/users',
    '/groups',
    '/auth',
    '/health',
    '/topics',
    '/articles',
    '/me',
    '/attempts',
    '/catalog',
    '/generation-jobs',
  ]

  const hit = prefixes.some((p) => path === p || path.startsWith(`${p}/`))
  if (!hit) {
    return
  }

  const target = `http://127.0.0.1:8080${path}${getRequestURL(event).search}`
  return proxyRequest(event, target)
})
