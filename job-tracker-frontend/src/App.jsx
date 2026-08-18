import { Routes, Route } from 'react-router-dom';
import ProtectedRoute from './components/ProtectedRoute';
import LoginPage from './pages/LoginPage';
import SignupPage from './pages/SignupPage';
import DiscoveredPage from './pages/DiscoveredPage';
import JobTrackerApp from './JobTrackerApp';

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignupPage />} />
      <Route
        path="/discovered"
        element={
          <ProtectedRoute>
            <DiscoveredPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/*"
        element={
          <ProtectedRoute>
            <JobTrackerApp />
          </ProtectedRoute>
        }
      />
    </Routes>
  );
}

export default App;
