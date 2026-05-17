import http from '../http'
import type { NotificationPreference, NotificationSearchReq } from '../interface'

export const getNotificationSummary = () => http.get('/notifications/summary')

export const getRecentNotifications = () => http.get('/notifications/recent')

export const getNotificationPreference = () => http.get('/notifications/preference')

export const updateNotificationPreference = (data: NotificationPreference) => {
  return http.post('/notifications/preference', data)
}

export const searchNotifications = (data: NotificationSearchReq) => {
  return http.post('/notifications/search', data)
}

export const markNotificationsRead = (data: { ids: number[] }) => {
  return http.post('/notifications/read', data)
}

export const markAllNotificationsRead = () => {
  return http.post('/notifications/read-all')
}

export const clearReadNotifications = () => {
  return http.post('/notifications/read/clear')
}

export const deleteNotification = (data: { id: number }) => {
  return http.post('/notifications/del', data)
}
