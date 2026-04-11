import { Link } from 'react-router-dom'
import ThemeToggle from './ThemeToggle'
import logo from '../logo.png'

interface HeaderProps {
  title: string
}

function Header({ title }: HeaderProps) {
  return (
    <header className="page-header">
      <div className="header-content">
        <div className="header-title">
          <Link to="/" className="logo-icon">
            <img src={logo} alt="Logo" />
          </Link>
          <h1>{title}</h1>
        </div>
        <div className="header-right">
          <Link to="/about" className="header-link">About</Link>
          <Link to="/license" className="header-link">Licenses</Link>
          <ThemeToggle />
        </div>
      </div>
    </header>
  )
}

export default Header
