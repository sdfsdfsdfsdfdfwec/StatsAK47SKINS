import React from 'react';

const RARITY_MAP = {
  'Covert': 'covert',
  'Classified': 'classified',
  'Restricted': 'restricted',
  'Mil-Spec': 'milspec',
  'Industrial Grade': 'milspec',
  'Consumer Grade': 'consumer',
  'Extraordinary': 'covert',
  'Exotic': 'classified',
  'High Grade': 'restricted',
  'Distinguished': 'milspec',
  'Exceptional': 'classified',
  'Superior': 'covert',
  'Master:': 'covert',
};

export default function SkinBadge({ skinName, status, rarity }) {
  const rarityLower = (rarity || '').toLowerCase();
  let badgeClass = 'badge-consumer';

  if (rarityLower.includes('covert') || rarityLower.includes('extraordinary') || rarityLower.includes('superior') || rarityLower.includes('master')) {
    badgeClass = 'badge-covert';
  } else if (rarityLower.includes('classified') || rarityLower.includes('exotic') || rarityLower.includes('exceptional')) {
    badgeClass = 'badge-classified';
  } else if (rarityLower.includes('restricted') || rarityLower.includes('high grade') || rarityLower.includes('distinguished')) {
    badgeClass = 'badge-restricted';
  } else if (rarityLower.includes('mil-spec') || rarityLower.includes('industrial') || rarityLower.includes('milspec')) {
    badgeClass = 'badge-milspec';
  }

  if (status && status.toLowerCase().includes('equipped')) {
    badgeClass += ' badge-equipped';
  }

  return (
    <span className={`badge ${badgeClass}`}>
      {skinName}
    </span>
  );
}
