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
        const [playerRes, statsRes, skinsRes, changesRes] = await Promise.all([
          fetchJSON(`/players/${steamid}`),
          fetchJSON(`/players/${steamid}/stats`),
          fetchJSON(`/players/${steamid}/skins`),
          fetchJSON(`/players/${steamid}/skin-changes`),
        ]);
        setPlayer(playerRes);
        setStats(statsRes.stats || []);
        setSkins(skinsRes.skins || []);
        setChanges(changesRes.changes || []);
      } catch (e) {
        setError(e.message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [steamid]);

  if (loading) return <div className="loading"><div className="spinner" /><p>Loading player profile...</p></div>;
  if (error) return <div className="error-msg"><p>{error}</p></div>;

  const p = player?.player || {};
  const latest = player?.latest_stats || {};
  const avatar = p.avatar_url || `https://avatars.steamstatic.com/${steamid}.jpg`;

  const tabs = [
    { key: 'skins', label: `Skins (${skins.length})` },
    { key: 'changes', label: `Changes (${changes.length})` },
    { key: 'alerts', label: 'Alerts' },
  ];

  return (
    <div>
      <Link to="/players" style={{ fontSize: '0.85rem', color: 'var(--text-muted)', display: 'inline-block', marginBottom: 16 }}>
        &larr; Back to Players
      </Link>

      <div className="profile-header">
        <img
          src={avatar}
          alt={p.player_name}
          className="profile-avatar"
          onError={(e) => { e.target.src = `https://ui-avatars.com/api/?name=${encodeURIComponent(p.player_name || '?')}&background=1e40af&color=fff&size=96`; }}
        />
        <div>
          <div className="profile-name">{p.player_name || 'Unknown'}</div>
          <div className="profile-steamid">{steamid}</div>
          <div className="profile-stats-row">
            <div>
              <div className="profile-stat-label">Position</div>
              <div className="profile-stat-value">#{latest.position || '—'}</div>
            </div>
            <div>
              <div className="profile-stat-label">Total Value</div>
              <div className="profile-stat-value value-positive">
                {latest.total_value != null ? `$${Number(latest.total_value).toLocaleString()}` : '—'}
              </div>
            </div>
            <div>
              <div className="profile-stat-label">Skin Count</div>
              <div className="profile-stat-value">{latest.skin_count || skins.length}</div>
            </div>
          </div>
        </div>
      </div>

      <div style={{ marginBottom: 24 }}>
        <StatChart
          data={stats}
          dataKey="position"
          title="Position Over Time"
          yLabel="Position"
          inverted
        />
      </div>

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
                <tr><td colSpan={5} className="empty-msg">No skins recorded</td></tr>
              ) : (
                skins.map((s, i) => (
                  <tr key={i}>
                    <td><SkinBadge skinName={s.skin_name} rarity={s.rarity} /></td>
                    <td>
                      {s.status && (
                        <span className={`badge ${s.status.toLowerCase().includes('equipped') ? 'badge-equipped' : 'badge-consumer'}`}>
                          {s.status}
                        </span>
                      )}
                    </td>
                    <td className="value-positive" style={{ fontFamily: 'Orbitron, monospace' }}>
                      {s.price != null ? `$${Number(s.price).toLocaleString()}` : '—'}
                    </td>
                    <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                      {s.first_seen ? new Date(s.first_seen).toLocaleString() : '—'}
                    </td>
                    <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
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
            <div className="empty-msg">No skin changes recorded</div>
          ) : (
            <div className="timeline">
              {changes.map((c, i) => (
                <div key={i} className={`timeline-item ${c.event_type === 'added' ? 'alert' : ''}`}>
                  <div style={{ fontWeight: 600 }}>
                    <SkinBadge skinName={c.skin_name} />
                  </div>
                  <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: 4 }}>
                    <span className={`badge ${c.event_type === 'added' ? 'badge-covert' : 'badge-milspec'}`} style={{ marginRight: 8 }}>
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
              <div className="empty-msg" style={{ padding: 24 }}>
                No Nouveau Rouge alerts for this player
              </div>
            ) : (
              <div className="timeline" style={{ marginTop: 16 }}>
                {changes
                  .filter(c => c.skin_name && c.skin_name.includes('Nouveau Rouge'))
                  .map((c, i) => (
                    <div key={i} className="timeline-item alert">
                      <div style={{ fontWeight: 600, color: 'var(--accent-red)' }}>{c.event_type}</div>
                      <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: 4 }}>
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
