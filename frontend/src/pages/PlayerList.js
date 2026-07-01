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
              (p.name && p.name.toLowerCase().includes(q)) ||
              (p.steamid && p.steamid.includes(q))
          );
        }
        setPlayers(list);
        setTotal(res.pagination?.total || res.total || 0);
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
        <div style={{ textAlign: 'center', padding: '60px 20px', color: '#94a3b8' }}>Loading players...</div>
      ) : error ? (
        <div style={{ textAlign: 'center', padding: '40px', color: '#ef4444', background: 'rgba(239,68,68,0.1)', borderRadius: 12 }}>
          {error}
        </div>
      ) : players.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '60px', color: '#64748b' }}>No players found</div>
      ) : (
        <>
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>#</th>
                  <th>Player</th>
                  <th>Last Seen</th>
                </tr>
              </thead>
              <tbody>
                {players.map((p) => (
                  <tr key={p.steamid}>
                    <td style={{ fontFamily: 'Orbitron, monospace', color: '#3b82f6' }}>
                      —
                    </td>
                    <td>
                      <Link to={`/player/${p.steamid}`} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                        <div style={{
                          width: 32, height: 32, borderRadius: '50%',
                          background: '#1e40af', display: 'flex', alignItems: 'center',
                          justifyContent: 'center', color: '#fff', fontSize: '0.7rem', fontWeight: 700,
                          border: '1px solid #1e3a5f'
                        }}>
                          {(p.name || '?')[0].toUpperCase()}
                        </div>
                        <span>
                          {p.name || 'Unknown'}
                          <span style={{ display: 'block', fontSize: '0.7rem', color: '#64748b', fontFamily: 'monospace' }}>
                            {p.steamid}
                          </span>
                        </span>
                      </Link>
                    </td>
                    <td style={{ color: '#64748b', fontSize: '0.85rem' }}>
                      {p.last_seen ? new Date(p.last_seen).toLocaleString() : '—'}
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
