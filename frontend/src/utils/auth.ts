const TOKEN_KEY = 'token'
const REMEMBER_KEY = 'rememberLogin'

export const getToken = () => {
  return localStorage.getItem(TOKEN_KEY) || sessionStorage.getItem(TOKEN_KEY) || ''
}

export const setToken = (token: string, remember = false) => {
  removeToken()
  if (remember) {
    localStorage.setItem(TOKEN_KEY, token)
    localStorage.setItem(REMEMBER_KEY, 'true')
  } else {
    sessionStorage.setItem(TOKEN_KEY, token)
    localStorage.removeItem(REMEMBER_KEY)
  }
}

export const removeToken = () => {
  sessionStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REMEMBER_KEY)
}

export const getRememberLogin = () => {
  return localStorage.getItem(REMEMBER_KEY) === 'true'
}
