import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { fetchJSON } from '../api';

export default function SkinTracker() {
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [view, setView] = useState('table');

  useEffect(() => {
    async function load() {
      try {
        const res = await fetchJSON('/skin-tracker');
        setAlerts(res.alerts || []);
      } catch (e) {
        setError(e.message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  if (loading) return <div className="loading"><div className="spinner" /><p>Loading skin tracker...</p></div>;
  if (error) return <div className="error-msg"><p>{error}</p></div>;

  return (
    <div>
      <div className="flex-between" style={{ marginBottom: 24 }}>
        <div>
          <h1 style={{ fontSize: '1.8rem', fontWeight: 700, fontFamily: 'Orbitron, monospace', marginBottom: 4 }}>
            Skin Tracker
          </h1>
          <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>
            AK-47 | Nouveau Rouge acquisition alerts
          </p>
        </div>
        <div className="btn-group">
          <button
            className={`btn ${view === 'table' ? 'btn-primary' : ''}`}
            onClick={() => setView('table')}
          >
            Table
          </button>
          <button
            className={`btn ${view === 'timeline' ? 'btn-primary' : ''}`}
            onClick={() => setView('timeline')}
          >
            Timeline
          </button>
        </div>
      </div>

      <div className="stat-card" style={{ marginBottom: 24 }}>
        <div className="stat-card-label">Total Acquisitions</div>
        <div className="stat-card-value" style={{ color: 'var(--accent-red)' }}>
          {alerts.length.toLocaleString()}
        </div>
        <div className="stat-card-sub">Players who obtained the Nouveau Rouge</div>
      </div>

      {alerts.length === 0 ? (
        <div className="card">
          <div className="empty-msg">No alerts recorded yet</div>
        </div>
      ) : view === 'table' ? (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Player</th>
                <th>Steam ID</th>
                <th>Detected</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {alerts.map((a, i) => (
                <tr key={i}>
                  <td>
                    <Link to={`/player/${a.steamid}`} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                      <img
                        src={`https://avatars.steamstatic.com/${a.steamid}.jpg`}
                        alt=""
                        style={{ width: 28, height: 28, borderRadius: '50%', border: '1px solid var(--border)' }}
                        onError={(e) => { e.target.src = `https://ui-avatars.com/api/?name=${encodeURIComponent(a.player_name || '?')}&background=1e40af&color=fff&size=28`; }}
                      />
                      {a.player_name || 'Unknown'}
                    </Link>
                  </td>
                  <td style={{ fontFamily: 'monospace', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                    {a.steamid}
                  </td>
                  <td style={{ fontSize: '0.85rem' }}>
                    {a.detected_at ? new Date(a.detected_at).toLocaleString() : '—'}
                  </td>
                  <td>
                    {a.storage_status && (
                      <span className="badge badge-equipped">{a.storage_status}</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="card">
          <div className="timeline">
            {alerts.map((a, i) => (
              <div key={i} className="timeline-item alert">
                <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 4 }}>
                  <Link to={`/player/${a.steamid}`} style={{ fontWeight: 600, fontSize: '1rem' }}>
                    {a.player_name || a.steamid}
                  </Link>
                  {a.storage_status && (
                    <span className="badge badge-equipped">{a.storage_status}</span>
                  )}
                </div>
                <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                  {a.detected_at ? new Date(a.detected_at).toLocaleString() : 'Unknown time'}
                </div>
                <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', fontFamily: 'monospace', marginTop: 2 }}>
                  {a.steamid}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
