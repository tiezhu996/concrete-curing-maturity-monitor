export interface AuditLog {
  id: number
  actor_id: number
  actor_name: string
  actor_role: string
  request_id: string
  entity_type: string
  entity_id: number
  action: string
  before_json: string
  after_json: string
  metadata_json: string
  created_at: string
}
