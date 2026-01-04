import { useEffect, useState } from 'react'
import PlayerList from './PlayerList'
import AddPlayerForm from './AddPlayerForm'
import AddMatchForm from './AddMatchForm'
import RecentMatches from './RecentMatches'
import { ladderService, Player as ProtoPlayer } from './grpc/ladderService'
import { useAuth } from './AuthContext'
import { supabase } from './supabase'

function Dashboard() {
    const [players, setPlayers] = useState<ProtoPlayer[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [refreshTrigger, setRefreshTrigger] = useState(0)
    const { user, isAdmin } = useAuth()

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

    const mappedPlayers = players.map(p => ({
        id: p.getId(),
        name: p.getName(),
        rank: p.getRank(),
    }))

    const handleLogout = async () => {
        await supabase.auth.signOut()
    }

    return (
        <div className="App">
            <header className="App-header">
                <h1>Squash Ladder</h1>
                <div className="user-info">
                    <span>{user?.email}</span>
                    <button onClick={handleLogout} className="logout-btn">Logout</button>
                </div>
            </header>
            <main className="App-main">
                {error && <div className="error-banner">Error: {error}</div>}

                <div className="dashboard-grid">
                    <div className="left-column">
                        <section className="add-player-section">
                            {isAdmin && <AddPlayerForm onPlayerAdded={handleDataUpdate} />}
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

export default Dashboard
