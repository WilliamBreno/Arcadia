import { Route, Routes } from 'react-router-dom'

import { SiteLayout } from '@/components/layout/site-layout'
import Home from '@/pages/Home'

function App() {
  return (
    <Routes>
      <Route element={<SiteLayout />}>
        <Route path="/" element={<Home />} />
      </Route>
    </Routes>
  )
}

export default App
