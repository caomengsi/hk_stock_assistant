import { BrowserRouter, Routes, Route, Link } from 'react-router-dom'
import Home from './pages/Home'
import Summary from './pages/Summary'
import Prediction from './pages/Prediction'
import './App.css'

function Nav() {
  return (
    <nav className="nav">
      <Link to="/">首页</Link>
      <Link to="/summary">大盘总结</Link>
      <Link to="/prediction">个股预测</Link>
    </nav>
  )
}

function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <Nav />
        <main className="main">
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/summary" element={<Summary />} />
            <Route path="/prediction" element={<Prediction />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}

export default App
