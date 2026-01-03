import React, { useEffect, useState, useCallback } from 'react'
import { ladderService, MatchResult, Player, SetScore } from './grpc/ladderService'

interface RecentMatchesProps {
    players: Player[]
    refreshTrigger: number // Simple counter to trigger refresh
}

const RecentMatches: React.FC<RecentMatchesProps> = ({ players, refreshTrigger }) => {
    const [matches, setMatches] = useState<MatchResult[]>([])
    const [loading, setLoading] = useState(false)

    const fetchMatches = useCallback(async () => {
        try {
            setLoading(true)
            const results = await ladderService.listRecentMatches(20)
            setMatches(results)
        } catch (err) {
            console.error('Failed to fetch recent matches', err)
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        fetchMatches()
    }, [fetchMatches, refreshTrigger])

    const getPlayerName = (id: string) => {
        const player = players.find(p => p.getId() === id)
        return player ? player.getName() : id
    }

    const formatScore = (setScores: SetScore[]) => {
        return setScores.map(s => {
            const p1 = s.getChallengerDefault() ? 'D' : s.getChallengerPoints()
            const p2 = s.getDefenderDefault() ? 'D' : s.getDefenderPoints()
            return `${p1}-${p2}`
        }).join(', ')
    }

    const formatDate = (timestampMs: number) => {
        return new Date(timestampMs).toLocaleString()
    }

    if (loading && matches.length === 0) {
        return <div>Loading matches...</div>
    }

    return (
        <div className="recent-matches-container">
            <h3>Recent Matches</h3>
            {matches.length === 0 ? (
                <p>No matches recorded yet.</p>
            ) : (
                <table className="matches-table">
                    <thead>
                        <tr>
                            <th>Date</th>
                            <th>Challenger</th>
                            <th>Defender</th>
                            <th>Winner</th>
                            <th>Score</th>
                        </tr>
                    </thead>
                    <tbody>
                        {matches.map((match, idx) => {
                            const p1Name = getPlayerName(match.getChallengerId())
                            const p2Name = getPlayerName(match.getDefenderId())
                            const winnerName = getPlayerName(match.getWinnerId())

                            return (
                                <tr key={match.getTransactionId() || idx}>
                                    <td>{formatDate(match.getTimestampMs())}</td>
                                    <td>{p1Name}</td>
                                    <td>{p2Name}</td>
                                    <td className="winner-cell">{winnerName}</td>
                                    <td>{formatScore(match.getSetScoresList())}</td>
                                </tr>
                            )
                        })}
                    </tbody>
                </table>
            )}
        </div>
    )
}

export default RecentMatches
