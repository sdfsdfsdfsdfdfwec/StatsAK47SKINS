import React, { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { fetchJSON } from '../api';
import StatChart from '../components/StatChart';
import SkinBadge from '../components/SkinBadge';

export default function PlayerProfile() {
  const { steamid } = useParams();
  const [player, setPlayer] = useState(null);
  const [stats, setStats] = useState([]);
  const [skins, setSkins] = useState([]);
  const [changes, setChanges] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [activeTab, setActiveTab] = useState('skins');

  useEffect(() => {
    async function load() {
      try {
        const results = await Promise.allSettled([
          fetchJSON(`/players/${steamid}`),
          fetchJSON(`/players/${steamid}/stats`),
          fetchJSON(`/players/${steamid}/skins`),
          fetchJSON(`/players/${steamid}/skin-changes`),
        ]);
        if (results[0].status === 'fulfilled') setPlayer(results[0].value);
        if (results[1].status === 'fulfilled') setStats(results[1].value?.stats || []);
        if (results[2].status === 'fulfilled') setSkins(results[2].value?.skins || []);
        if (results[3].status === 'fulfilled') setChanges(results[3].value?.events || []);
      } catch (e) {
        setError(e.message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [steamid]);

  if (loading) return (
    <div style={{ textAlign: 'center', padding: '60px 20px', color: '#94a3b8' }}>
      Loading player profile...
    </div>
  );

  if (error) return (
    <div style={{ textAlign: 'center', padding: '40px', color: '#ef4444', background: 'rgba(239,68,68,0.1)', borderRadius: 12 }}>
      {error}
    </div>
  );

  const p = player?.player || player || {};
  const latest = player?.stats || {};
  const name = p.name || 'Unknown';

  const tabs = [
    { key: 'skins', label: `Skins (${skins.length})` },
    { key: 'changes', label: `Changes (${changes.length})` },
    { key: 'alerts', label: 'Alerts' },
  ];

  return (
    <div>
      <Link to="/players" style={{ fontSize: '0.85rem', color: '#64748b', display: 'inline-block', marginBottom: 16 }}>
        &larr; Back to Players
      </Link>

      <div className="profile-header">
        <div style={{
          width: 96, height: 96, borderRadius: '50%', background: '#1e40af',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#fff', fontSize: '2rem', fontWeight: 700, border: '3px solid #3b82f6',
          boxShadow: '0 0 20px rgba(59,130,246,0.3)', flexShrink: 0
        }}>
          {name[0].toUpperCase()}
        </div>
        <div>
          <div className="profile-name">{name}</div>
          <div className="profile-steamid">{steamid}</div>
          <div className="profile-stats-row">
            {latest.position != null && (
              <div>
                <div className="profile-stat-label">Position</div>
                <div className="profile-stat-value">#{latest.position}</div>
              </div>
            )}
            {latest.total_value != null && (
              <div>
                <div className="profile-stat-label">Total Value</div>
                <div className="profile-stat-value" style={{ color: '#10b981' }}>
                  ${Number(latest.total_value).toLocaleString()}
                </div>
              </div>
            )}
            <div>
              <div className="profile-stat-label">Skins</div>
              <div className="profile-stat-value">{latest.skin_count || skins.length}</div>
            </div>
          </div>
        </div>
      </div>

      {stats.length > 0 && (
        <div style={{ marginBottom: 24 }}>
          <StatChart data={stats} dataKey="position" title="Position Over Time" yLabel="Position" inverted />
        </div>
      )}

      <div style={{ display: 'flex', gap: 8, marginBottom: 20 }}>
        {tabs.map((t) => (
          <button
            key={t.key}
            className={`btn ${activeTab === t.key ? 'btn-primary' : ''}`}
            onClick={() => setActiveTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {activeTab === 'skins' && (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Skin</th>
                <th>Status</th>
                <th>Price</th>
                <th>First Seen</th>
                <th>Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {skins.length === 0 ? (
                <tr><td colSpan={5} style={{ textAlign: 'center', padding: 40, color: '#64748b' }}>No skins recorded</td></tr>
              ) : (
                skins.map((s, i) => (
                  <tr key={i}>
                    <td><SkinBadge skinName={s.skin_name} /></td>
                    <td>
                      {s.status && (
                        <span className={`badge ${s.status === 'stored' ? 'badge-equipped' : 'badge-consumer'}`}>
                          {s.status}
                        </span>
                      )}
                    </td>
                    <td style={{ color: '#10b981', fontFamily: 'Orbitron, monospace' }}>
                      {s.price != null ? `$${Number(s.price).toLocaleString()}` : '—'}
                    </td>
                    <td style={{ fontSize: '0.8rem', color: '#64748b' }}>
                      {s.first_seen ? new Date(s.first_seen).toLocaleString() : '—'}
                    </td>
                    <td style={{ fontSize: '0.8rem', color: '#64748b' }}>
                      {s.last_seen ? new Date(s.last_seen).toLocaleString() : '—'}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'changes' && (
        <div className="card">
          {changes.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '40px', color: '#64748b' }}>No skin changes recorded</div>
          ) : (
            <div className="timeline">
              {changes.map((c, i) => (
                <div key={i} className={`timeline-item ${c.event_type === 'skin_added' ? 'alert' : ''}`}>
                  <div style={{ fontWeight: 600 }}>
                    <SkinBadge skinName={c.skin_name} />
                  </div>
                  <div style={{ fontSize: '0.8rem', color: '#64748b', marginTop: 4 }}>
                    <span className={`badge ${c.event_type === 'skin_added' ? 'badge-covert' : 'badge-milspec'}`} style={{ marginRight: 8 }}>
                      {c.event_type}
                    </span>
                    {c.detected_at ? new Date(c.detected_at).toLocaleString() : 'Unknown time'}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {activeTab === 'alerts' && (
        <div className="card">
          <SkinBadge skinName="AK-47 | Nouveau Rouge" rarity="Covert" />
          <div style={{ marginTop: 16 }}>
            {changes.filter(c => c.skin_name && c.skin_name.includes('Nouveau Rouge')).length === 0 ? (
              <div style={{ textAlign: 'center', padding: '24px', color: '#64748b' }}>
                No Nouveau Rouge alerts for this player
              </div>
            ) : (
              <div className="timeline" style={{ marginTop: 16 }}>
                {changes
                  .filter(c => c.skin_name && c.skin_name.includes('Nouveau Rouge'))
                  .map((c, i) => (
                    <div key={i} className="timeline-item alert">
                      <div style={{ fontWeight: 600, color: '#ef4444' }}>{c.event_type}</div>
                      <div style={{ fontSize: '0.8rem', color: '#64748b', marginTop: 4 }}>
                        {c.detected_at ? new Date(c.detected_at).toLocaleString() : 'Unknown time'}
                      </div>
                    </div>
                  ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
