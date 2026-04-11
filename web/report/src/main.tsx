import { render } from 'react-dom'
import { HashRouter, Route, Routes, useParams } from 'react-router-dom'
import 'devicon/devicon-base.css'
import './fonts.css'
import App from './App.tsx'
import ImageDetail from './components/ImageDetail.tsx'
import { ThemeProvider } from './ThemeContext.tsx'
import { getReportData } from './mockData.ts'

declare global {
  interface Window {
    __REPORT_DATA__?: any;
  }
}

const data = getReportData()

function WrapImageDetail() {
  const { imageName, kind } = useParams()
  return <ImageDetail data={data} imageName={imageName} kind={kind} />
}

function WrapApp() {
  return <App data={data} />
}

render(
  <ThemeProvider>
    <HashRouter>
      <Routes>
        <Route path="/" element={<WrapApp />} />
        <Route path="/image/:imageName/:kind" element={<WrapImageDetail />} />
        <Route path="/image/:imageName" element={<WrapImageDetail />} />
      </Routes>
    </HashRouter>
  </ThemeProvider>,
  document.getElementById('root')
)