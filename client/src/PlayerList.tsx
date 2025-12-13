import React from 'react'
import './PlayerList.css'

interface Player {
  id: number
  name: string
  rank: number
}

interface PlayerListProps {
  players: Player[]
}

const PlayerList: React.FC<PlayerListProps> = ({ players }) => {
  if (players.length === 0) {
    return <p>No players found.</p>
  }

  return (
    <div className="player-list">
      <h2>Current Rankings</h2>
      <div className="player-table">
        <div className="player-header">
          <div className="rank-col">Rank</div>
          <div className="name-col">Player Name</div>
        </div>
        {players.map((player) => (
          <div key={player.id} className="player-row">
            <div className="rank-col">
              <span className="rank-badge">{player.rank}</span>
            </div>
            <div className="name-col">{player.name}</div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default PlayerList

