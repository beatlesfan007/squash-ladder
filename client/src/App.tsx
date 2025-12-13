import React, { useEffect, useState } from 'react'
import PlayerList from './PlayerList'
import { playersService, Player } from './grpc/playersService'
import './App.css'

function App() {
  const [players, setPlayers] = useState<Player[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchPlayers()
  }, [])

  const fetchPlayers = async () => {
    try {
      setLoading(true)
      const response = await playersService.listPlayers()
      // Convert proto Player messages to plain objects
      const playersList = response.getPlayersList().map((p: Player) => ({
        id: p.getId(),
        name: p.getName(),
        rank: p.getRank(),
      }))
      setPlayers(playersList)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch players')
      console.error('Error fetching players:', err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="App">
      <header className="App-header">
        <h1>Squash Ladder</h1>
      </header>
      <main className="App-main">
        {loading && <p>Loading players...</p>}
        {error && <p className="error">Error: {error}</p>}
        {!loading && !error && <PlayerList players={players} />}
      </main>
    </div>
  )
}

export default App

