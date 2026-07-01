const BASE_URL = process.env.REACT_APP_API_URL || '/api';

export async function fetchJSON(endpoint, options = {}) {
  const url = `${BASE_URL}${endpoint}`;
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`API error ${res.status}: ${text || res.statusText}`);
  }
  const json = await res.json();
  if (json && json.data !== undefined) {
    return json.data;
  }
  return json;
}
