import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import { ThemeProvider } from './context/ThemeContext';
import { WebSocketProvider } from './context/WebSocketContext';
import { ToastContainer } from './components/Toast';
import Dashboard from './pages/Dashboard';
import ServerDetail from './pages/ServerDetail';
import Login from './pages/Login';
import Settings from './pages/Settings';
import OAuthCallback from './pages/OAuthCallback';

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <WebSocketProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/server/:id" element={<ServerDetail />} />
              <Route path="/login" element={<Login />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="/oauth-callback" element={<OAuthCallback />} />
            </Routes>
            <ToastContainer />
          </BrowserRouter>
        </WebSocketProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
