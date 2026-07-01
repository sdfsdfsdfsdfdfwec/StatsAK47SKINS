import React from 'react';
import { Link } from 'react-router-dom';

export default function PlayerCard({ player }) {
  const { steamid, player_name, total_value, position, skin_count, avatar_url } = player;
  const avatar = avatar_url || `https://avatars.steamstatic.com/${steamid}.jpg`;

  return (
    <Link to={`/player/${steamid}`} className="player-card">
      <img
        src={avatar}
        alt={player_name}
        className="player-avatar"
        onError={(e) => { e.target.src = `https://ui-avatars.com/api/?name=${encodeURIComponent(player_name)}&background=1e40af&color=fff&size=56`; }}
      />
      <div className="player-card-info">
        <div className="player-card-name">{player_name || 'Unknown'}</div>
        <div className="player-card-steamid">{steamid}</div>
      </div>
      <div className="player-card-stats">
        <div className="player-card-stat">
          <div className="player-card-stat-label">Position</div>
          <div className="player-card-stat-value">#{position || '—'}</div>
        </div>
        <div className="player-card-stat">
          <div className="player-card-stat-label">Value</div>
          <div className="player-card-stat-value value-positive">
            {total_value != null ? `$${Number(total_value).toLocaleString()}` : '—'}
          </div>
        </div>
        <div className="player-card-stat">
          <div className="player-card-stat-label">Skins</div>
          <div className="player-card-stat-value">{skin_count || 0}</div>
        </div>
      </div>
    </Link>
  );
}
