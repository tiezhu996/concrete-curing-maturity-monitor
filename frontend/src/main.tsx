import React from 'react'
import ReactDOM from 'react-dom/client'
import { App as AntdApp, ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { RouterProvider } from 'react-router-dom'
import { router } from './router'
import './app.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider locale={zhCN} theme={{
      token: {
        colorPrimary: '#2f6b4f', colorInfo: '#476b89', colorWarning: '#c27a24', colorError: '#a34c3d',
        borderRadius: 4, fontFamily: 'Inter, "Noto Sans SC", "PingFang SC", system-ui, sans-serif',
        colorBgLayout: '#f4f5f2', colorText: '#242a27', colorBorder: '#d9ddd7', controlHeight: 36,
      },
      components: { Table: { headerBg: '#eef0ec', headerColor: '#4b544f' }, Menu: { itemBorderRadius: 3 } },
    }}>
      <AntdApp><RouterProvider router={router} /></AntdApp>
    </ConfigProvider>
  </React.StrictMode>,
)
