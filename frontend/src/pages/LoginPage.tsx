import { Alert, Button, Form, Input } from 'antd'
import { ClipboardCheck, LockKeyhole, UserRound } from 'lucide-react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import { errorMessage } from '../api/client'
import { SafetyNotice } from '../components/common/SafetyNotice'

export function LoginPage() {
  const { authenticated, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  if (authenticated) return <Navigate to="/sections" replace />

  const submit = async (values: { username: string; password: string }) => {
    setLoading(true); setError('')
    try {
      await login(values.username, values.password)
      const target = (location.state as { from?: string } | null)?.from ?? '/sections'
      navigate(target, { replace: true })
    } catch (reason) {
      setError(errorMessage(reason))
    } finally { setLoading(false) }
  }

  return (
    <main className="login-page">
      <section className="login-identity">
        <div className="login-brand"><ClipboardCheck size={30} /><span>CCMM / 01</span></div>
        <div>
          <span className="eyebrow">CONCRETE CURING RECORD</span>
          <h1>混凝土养护<br />成熟度监测</h1>
          <p>温度序列、配合比标定和工程签认的可追溯工作台。</p>
        </div>
        <div className="worksheet-lines"><span>FORMULA</span><strong>M = Σ(Tavg − T0) × Δt</strong></div>
      </section>
      <section className="login-form-panel">
        <div className="login-form-wrap">
          <span className="eyebrow">AUTHORIZED ACCESS</span>
          <h2>进入工程工作台</h2>
          <p>使用分配的试验室、现场或审核账号。</p>
          {error && <Alert type="error" showIcon message={error} className="form-alert" />}
          <Form layout="vertical" size="large" initialValues={{ username: 'admin', password: 'admin123' }} onFinish={submit}>
            <Form.Item label="账号" name="username" rules={[{ required: true, message: '请输入账号' }]}>
              <Input prefix={<UserRound size={17} />} autoComplete="username" />
            </Form.Item>
            <Form.Item label="密码" name="password" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password prefix={<LockKeyhole size={17} />} autoComplete="current-password" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>登录</Button>
          </Form>
          <SafetyNotice compact />
        </div>
      </section>
    </main>
  )
}
