import React from 'react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Area, AreaChart
} from 'recharts';

function formatDate(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export default function StatChart({ data, dataKey = 'position', title, yLabel, inverted = false }) {
  if (!data || data.length === 0) {
    return (
      <div className="chart-container">
        {title && <h3 className="section-title">{title}</h3>}
        <div className="empty-msg">No data available</div>
      </div>
    );
  }

  const chartData = data.map((item, i) => ({
    ...item,
    label: formatDate(item.snapshot_time),
    value: Number(item[dataKey]) || 0,
  }));

  return (
    <div className="chart-container">
      {title && <h3 className="section-title">{title}</h3>}
      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={chartData} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#1e3a5f" opacity={0.3} />
          <XAxis
            dataKey="label"
            stroke="#64748b"
            tick={{ fontSize: 12 }}
            tickLine={false}
          />
          <YAxis
            stroke="#64748b"
            tick={{ fontSize: 12 }}
            tickLine={false}
            reversed={inverted}
            label={yLabel ? { value: yLabel, angle: -90, position: 'insideLeft', fill: '#64748b' } : undefined}
          />
          <Tooltip
            contentStyle={{
              background: '#1a2235',
              border: '1px solid #1e3a5f',
              borderRadius: 8,
              color: '#e2e8f0',
              fontSize: 13,
            }}
            labelStyle={{ color: '#94a3b8' }}
          />
          <Area
            type="monotone"
            dataKey="value"
            stroke="#3b82f6"
            strokeWidth={2}
            fill="url(#colorValue)"
            dot={{ fill: '#3b82f6', r: 4, strokeWidth: 0 }}
            activeDot={{ r: 6, fill: '#06b6d4' }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
