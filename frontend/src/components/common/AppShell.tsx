import { useState } from 'react'
import { Avatar, Button, Drawer, Dropdown, Grid, Layout, Menu, type MenuProps } from 'antd'
import { Activity, Beaker, ChevronDown, ClipboardList, FileClock, LogOut, Menu as MenuIcon, Thermometer, Waves } from 'lucide-react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../../hooks/useAuth'
import { can } from '../../utils/permissions'
import { SafetyNotice } from './SafetyNotice'

const { Header, Sider, Content } = Layout

export function AppShell() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuth()
  const screens = Grid.useBreakpoint()
  const mobile = !screens.lg
  const [drawerOpen, setDrawerOpen] = useState(false)

  const navigation: MenuProps['items'] = [
    { key: '/sections', icon: <Waves size={17} />, label: '浇筑区段' },
    { key: '/mix-designs', icon: <Beaker size={17} />, label: '配合比版本' },
    { key: '/temperatures', icon: <Thermometer size={17} />, label: '温度序列' },
    { key: '/forecasts', icon: <Activity size={17} />, label: '成熟度预测' },
    ...(can(user?.role, 'audit:read') ? [{ key: '/audit', icon: <FileClock size={17} />, label: '审计中心' }] : []),
  ]

  const menu = (
    <Menu
      mode="inline"
      selectedKeys={[location.pathname]}
      items={navigation}
      onClick={({ key }) => { navigate(key); setDrawerOpen(false) }}
    />
  )

  const accountItems: MenuProps['items'] = [
    { key: 'role', disabled: true, label: user?.role ?? '' },
    { type: 'divider' },
    { key: 'logout', icon: <LogOut size={16} />, label: '退出登录', onClick: () => { logout(); navigate('/login') } },
  ]

  return (
    <Layout className="app-layout">
      {!mobile && (
        <Sider width={232} theme="light" className="app-sider">
          <div className="brand-block">
            <div className="brand-mark"><ClipboardList size={22} /></div>
            <div><strong>养护成熟度</strong><span>工程试验工作台</span></div>
          </div>
          <div className="sider-context"><span>项目状态</span><strong><i /> 离线仿真</strong></div>
          {menu}
          <div className="sider-footer">Nurse-Saul · v1.0</div>
        </Sider>
      )}
      <Layout>
        <Header className="app-header">
          <div className="mobile-brand">
            {mobile && <Button type="text" icon={<MenuIcon size={20} />} onClick={() => setDrawerOpen(true)} aria-label="打开导航" />}
            {mobile && <strong>养护成熟度</strong>}
          </div>
          <Dropdown menu={{ items: accountItems }} placement="bottomRight">
            <Button type="text" className="account-button">
              <Avatar size={28}>{user?.display_name?.slice(0, 1)}</Avatar>
              {!mobile && <span>{user?.display_name}</span>}
              <ChevronDown size={14} />
            </Button>
          </Dropdown>
        </Header>
        <SafetyNotice compact />
        <Content className="app-content"><Outlet /></Content>
      </Layout>
      <Drawer title="功能导航" placement="left" width={286} open={drawerOpen} onClose={() => setDrawerOpen(false)} className="nav-drawer">{menu}</Drawer>
    </Layout>
  )
}
