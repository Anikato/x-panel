import http from '../http'
import type { NotificationSearchReq } from '../interface'

export const getNotificationSummary = () => http.get('/notifications/summary')

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
