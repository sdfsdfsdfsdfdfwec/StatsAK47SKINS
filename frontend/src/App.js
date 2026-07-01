import React from 'react';
import { Routes, Route } from 'react-router-dom';
import Navbar from './components/Navbar';
import Dashboard from './pages/Dashboard';
import PlayerList from './pages/PlayerList';
import PlayerProfile from './pages/PlayerProfile';
import SkinTracker from './pages/SkinTracker';

export default function App() {
  return (
    <div className="app-layout">
      <Navbar />
      <main className="app-content">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/players" element={<PlayerList />} />
          <Route path="/player/:steamid" element={<PlayerProfile />} />
          <Route path="/skin-tracker" element={<SkinTracker />} />
        </Routes>
      </main>
    </div>
  );
}
