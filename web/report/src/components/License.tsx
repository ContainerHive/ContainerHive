import { Link } from 'react-router-dom'
import noticeContent from '../NOTICE?raw'

function License() {
  return (
    <div className="container">
      <Link to="/" className="back-link">← Back to Gallery</Link>
      <pre className="notice-block"><code>{noticeContent}</code></pre>
    </div>
  )
}

export default License
