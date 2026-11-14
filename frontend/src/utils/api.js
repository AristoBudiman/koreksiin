const TOKEN_KEY = 'koreksiin_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
}

export async function apiFetch(path, options = {}) {
  const token = getToken()
  const headers = {
    ...(options.headers || {})
  }

  if (!(options.body instanceof FormData) && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(path, {
    ...options,
    headers
  })

  const contentType = res.headers.get('content-type') || ''
  let data = null
  if (contentType.includes('application/json')) {
    data = await res.json()
  }

  if (!res.ok) {
    const errMessage = data?.error || `Permintaan gagal dengan kode ${res.status}`
    const err = new Error(errMessage)
    err.status = res.status
    err.data = data
    throw err
  }

  return data
}

