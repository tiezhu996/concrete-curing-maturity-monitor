export interface PageData<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface ApiEnvelope<T> {
  code: string
  message: string
  data: T
  request_id: string
}

export interface QueryParams {
  page?: number
  page_size?: number
  [key: string]: string | number | boolean | undefined
}
