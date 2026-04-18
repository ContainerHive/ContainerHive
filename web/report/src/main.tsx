import { render } from 'react-dom'
import { HashRouter, Route, Routes, useParams, useLocation } from 'react-router-dom'
import 'devicon/devicon-base.css'
import './fonts.css'
import Header from './components/Header.tsx'
import App from './App.tsx'
import ImageDetail from './components/ImageDetail.tsx'
import License from './components/License.tsx'
import About from './components/About.tsx'
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

function AppTitle() {
  const location = useLocation()
  const path = location.pathname
  
  if (path.startsWith('/image/')) {
    return <Header title="Image Details" />
  }
  if (path === '/about') {
    return <Header title="About" />
  }
  if (path === '/license') {
    return <Header title="Licenses" />
  }
  return <Header title="Image Overview" />
}

render(
  <ThemeProvider>
    <HashRouter>
      <AppTitle />
      <Routes>
        <Route path="/" element={<WrapApp />} />
        <Route path="/image/:imageName/:kind" element={<WrapImageDetail />} />
        <Route path="/image/:imageName" element={<WrapImageDetail />} />
        <Route path="/license" element={<License />} />
        <Route path="/about" element={<About />} />
      </Routes>
    </HashRouter>
  </ThemeProvider>,
  document.getElementById('root')
)
