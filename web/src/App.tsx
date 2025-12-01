import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import { ThemeProvider } from './context/ThemeContext';
import Dashboard from './pages/Dashboard';
import ServerDetail from './pages/ServerDetail';
import Login from './pages/Login';
import Settings from './pages/Settings';

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/server/:id" element={<ServerDetail />} />
            <Route path="/login" element={<Login />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
