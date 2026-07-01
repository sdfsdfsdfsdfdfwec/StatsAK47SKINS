import React, { useState, useEffect } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { fetchJSON } from '../api';

export default function PlayerList() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [players, setPlayers] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const page = parseInt(searchParams.get('page') || '1', 10);
  const limit = parseInt(searchParams.get('limit') || '50', 10);
  const search = searchParams.get('search') || '';
  const [searchInput, setSearchInput] = useState(search);

  useEffect(() => {
    async function load() {
      setLoading(true);
      try {
        const params = new URLSearchParams({ page, limit });
        const res = await fetchJSON(`/players?${params}`);
        let list = res.players || [];
        if (search) {
          const q = search.toLowerCase();
          list = list.filter(
            (p) =>
              (p.player_name && p.player_name.toLowerCase().includes(q)) ||
              (p.steamid && p.steamid.includes(q))
          );
        }
        setPlayers(list);
        setTotal(res.total || 0);
      } catch (e) {
        setError(e.message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [page, limit, search]);

  const totalPages = Math.ceil(total / limit);

  const goToPage = (p) => {
    const params = new URLSearchParams(searchParams);
    params.set('page', p);
    setSearchParams(params);
  };

  const handleSearch = (e) => {
    e.preventDefault();
    const params = new URLSearchParams(searchParams);
    if (searchInput.trim()) {
      params.set('search', searchInput.trim());
    } else {
      params.delete('search');
    }
    params.set('page', '1');
    setSearchParams(params);
  };

  return (
    <div>
      <h1 style={{ fontSize: '1.8rem', fontWeight: 700, marginBottom: 24, fontFamily: 'Orbitron, monospace' }}>
        Players
      </h1>

      <div className="card" style={{ marginBottom: 24 }}>
        <form onSubmit={handleSearch} style={{ display: 'flex', gap: 12 }}>
          <input
            type="text"
            className="search-input"
            placeholder="Filter by name or Steam ID..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
          />
          <button type="submit" className="btn btn-primary">Filter</button>
        </form>
      </div>

      {loading ? (
        <div className="loading"><div className="spinner" /><p>Loading players...</p></div>
      ) : error ? (
        <div className="error-msg"><p>{error}</p></div>
      ) : players.length === 0 ? (
        <div className="empty-msg">No players found</div>
      ) : (
        <>
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>#</th>
                  <th>Player</th>
                  <th>Total Value</th>
                  <th>Skins</th>
                  <th>Last Snapshot</th>
                </tr>
              </thead>
              <tbody>
                {players.map((p) => (
                  <tr key={p.steamid}>
                    <td style={{ fontFamily: 'Orbitron, monospace', color: 'var(--accent-blue)' }}>
                      {p.position || '—'}
                    </td>
                    <td>
                      <Link to={`/player/${p.steamid}`} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                        <img
                          src={p.avatar_url || `https://avatars.steamstatic.com/${p.steamid}.jpg`}
                          alt=""
                          style={{ width: 32, height: 32, borderRadius: '50%', border: '1px solid var(--border)' }}
                          onError={(e) => { e.target.src = `https://ui-avatars.com/api/?name=${encodeURIComponent(p.player_name || '?')}&background=1e40af&color=fff&size=32`; }}
                        />
                        <span>
                          {p.player_name || 'Unknown'}
                          <span style={{ display: 'block', fontSize: '0.7rem', color: 'var(--text-muted)', fontFamily: 'monospace' }}>
                            {p.steamid}
                          </span>
                        </span>
                      </Link>
                    </td>
                    <td className="value-positive" style={{ fontFamily: 'Orbitron, monospace' }}>
                      {p.total_value != null ? `$${Number(p.total_value).toLocaleString()}` : '—'}
                    </td>
                    <td>{p.skin_count || 0}</td>
                    <td style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>
                      {p.last_snapshot ? new Date(p.last_snapshot).toLocaleString() : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="pagination">
              <button
                className="pagination-btn"
                disabled={page <= 1}
                onClick={() => goToPage(page - 1)}
              >
                Prev
              </button>
              {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
                let p;
                if (totalPages <= 7) {
                  p = i + 1;
                } else if (page <= 4) {
                  p = i + 1;
                } else if (page >= totalPages - 3) {
                  p = totalPages - 6 + i;
                } else {
                  p = page - 3 + i;
                }
                return (
                  <button
                    key={p}
                    className={`pagination-btn ${p === page ? 'active' : ''}`}
                    onClick={() => goToPage(p)}
                  >
                    {p}
                  </button>
                );
              })}
              <button
                className="pagination-btn"
                disabled={page >= totalPages}
                onClick={() => goToPage(page + 1)}
              >
                Next
              </button>
              <span className="pagination-info">
                Page {page} of {totalPages} ({total.toLocaleString()} players)
              </span>
            </div>
          )}
        </>
      )}
    </div>
  );
}
