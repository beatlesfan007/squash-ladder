import { useEffect, useState } from 'react'
import PlayerList from './PlayerList'
import AddPlayerForm from './AddPlayerForm'
import AddMatchForm from './AddMatchForm'
import RecentMatches from './RecentMatches'
import { ladderService, Player as ProtoPlayer } from './grpc/ladderService'
import './App.css'

function App() {
  const [players, setPlayers] = useState<ProtoPlayer[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  useEffect(() => {
    fetchPlayers()
  }, [refreshTrigger])

  const fetchPlayers = async () => {
    try {
      setLoading(true)
      const response = await ladderService.listPlayers()
      setPlayers(response.getPlayersList())
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch players')
      console.error('Error fetching players:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleDataUpdate = () => {
    setRefreshTrigger(prev => prev + 1)
  }

  // Convert proto players to the interface expected by PlayerList component
  // Or better yet, update PlayerList to use ProtoPlayer or just map it here.
  // PlayerList expects { id, name, rank }. ProtoPlayer has getters.
  const mappedPlayers = players.map(p => ({
    id: p.getId(),
    name: p.getName(),
    rank: p.getRank(),
  }))

  return (
    <div className="App">
      <header className="App-header">
        <h1>Squash Ladder</h1>
      </header>
      <main className="App-main">
        {error && <div className="error-banner">Error: {error}</div>}

        <div className="dashboard-grid">
          <div className="left-column">
            <section className="add-player-section">
              <AddPlayerForm onPlayerAdded={handleDataUpdate} />
            </section>
            <section className="ladder-section">
              {loading ? <p>Loading ladder...</p> : <PlayerList players={mappedPlayers} />}
            </section>
          </div>

          <div className="right-column">
            <section className="add-match-section">
              <AddMatchForm players={players} onMatchAdded={handleDataUpdate} />
            </section>
            <section className="recent-matches-section">
              <RecentMatches players={players} refreshTrigger={refreshTrigger} />
            </section>
          </div>
        </div>
      </main>
    </div>
  )
}

export default App

