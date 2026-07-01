import React from 'react';
import { NavLink } from 'react-router-dom';

export default function Navbar() {
  return (
    <nav className="navbar">
      <div className="navbar-logo">STATSAK47</div>
      <div className="navbar-links">
        <NavLink to="/" end className={({ isActive }) => isActive ? 'active' : ''}>
          Dashboard
        </NavLink>
        <NavLink to="/players" className={({ isActive }) => isActive ? 'active' : ''}>
          Players
        </NavLink>
        <NavLink to="/skin-tracker" className={({ isActive }) => isActive ? 'active' : ''}>
          Skin Tracker
        </NavLink>
      </div>
    </nav>
  );
}
