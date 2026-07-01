import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { fetchJSON } from '../api';

export default function Dashboard() {
  const [stats, setStats] = useState(null);
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [search, setSearch] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    async function load() {
      try {
        const [statsRes, alertsRes] = await Promise.allSettled([
          fetchJSON('/stats'),
          fetchJSON('/skin-tracker'),
        ]);
        if (statsRes.status === 'fulfilled') setStats(statsRes.value);
        if (alertsRes.status === 'fulfilled') setAlerts(alertsRes.value?.alerts || []);
      } catch (e) {
        setError(e.message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  const handleSearch = (e) => {
    e.preventDefault();
    if (search.trim()) {
      navigate(`/players?search=${encodeURIComponent(search.trim())}`);
    }
  };

  if (loading) return (
    <div style={{ textAlign: 'center', padding: '60px 20px' }}>
      <div style={{
        width: 40, height: 40, border: '3px solid #1e3a5f',
        borderTopColor: '#3b82f6', borderRadius: '50%',
        animation: 'spin 0.8s linear infinite', margin: '0 auto 16px'
      }} />
      <p style={{ color: '#94a3b8' }}>Loading dashboard...</p>
    </div>
  );

  return (
    <div>
      <h1 style={{ fontSize: '1.8rem', fontWeight: 700, marginBottom: 24, fontFamily: 'Orbitron, monospace' }}>
        Dashboard
      </h1>

      <div className="card-grid" style={{ marginBottom: 32 }}>
        <div className="stat-card">
          <div className="stat-card-label">Total Players</div>
          <div className="stat-card-value">{stats?.total_players?.toLocaleString() || '0'}</div>
          <div className="stat-card-sub">Tracked on leaderboard</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-label">Total Snapshots</div>
          <div className="stat-card-value">{stats?.total_snapshots?.toLocaleString() || '0'}</div>
          <div className="stat-card-sub">Data points collected</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-label">Nouveau Rouge Alerts</div>
          <div className="stat-card-value" style={{ color: '#ef4444' }}>
            {stats?.total_alerts?.toLocaleString() || '0'}
          </div>
          <div className="stat-card-sub">AK47 skin acquisitions</div>
        </div>
      </div>

      {error && (
        <div style={{
          textAlign: 'center', padding: '20px', color: '#f59e0b',
          background: 'rgba(245,158,11,0.1)', border: '1px solid rgba(245,158,11,0.2)',
          borderRadius: 12, marginBottom: 24, fontSize: '0.9rem'
        }}>
          API unavailable - some data may not be loaded yet. Backend is starting up...
        </div>
      )}

      <div className="card" style={{ marginBottom: 32 }}>
        <h2 className="section-title">Quick Search</h2>
        <form onSubmit={handleSearch} style={{ display: 'flex', gap: 12 }}>
          <input
            type="text"
            className="search-input"
            placeholder="Search by player name or Steam ID..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <button type="submit" className="btn btn-primary">Search</button>
        </form>
      </div>

      <div className="card">
        <div className="flex-between" style={{ marginBottom: 16 }}>
          <h2 className="section-title" style={{ marginBottom: 0 }}>Recent Nouveau Rouge Alerts</h2>
          <Link to="/skin-tracker" className="btn">View All</Link>
        </div>
        {alerts.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '40px', color: '#64748b' }}>
            No alerts yet — data is being collected...
          </div>
        ) : (
          alerts.slice(0, 10).map((alert, i) => (
            <div key={i} className="alert-item">
              <div className="alert-dot" />
              <div className="alert-info">
                <div className="alert-player">
                  <Link to={`/player/${alert.steamid}`}>{alert.player_name || alert.steamid}</Link>
                </div>
                <div className="alert-time">
                  {alert.detected_at ? new Date(alert.detected_at).toLocaleString() : 'Unknown'}
                </div>
              </div>
              <div className="alert-status">
                <span className="badge badge-covert">Nouveau Rouge</span>
                {alert.storage_status && (
                  <span className="badge badge-equipped" style={{ marginLeft: 6 }}>{alert.storage_status}</span>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
