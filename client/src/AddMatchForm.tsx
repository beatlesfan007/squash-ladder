import React, { useState } from 'react'
import { ladderService, SetScore, Player } from './grpc/ladderService'

interface AddMatchFormProps {
    players: Player[]
    onMatchAdded: () => void
}

const AddMatchForm: React.FC<AddMatchFormProps> = ({ players, onMatchAdded }) => {
    const [player1Id, setPlayer1Id] = useState('')
    const [player2Id, setPlayer2Id] = useState('')
    const [winnerId, setWinnerId] = useState('')
    const [scoreInput, setScoreInput] = useState('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!player1Id || !player2Id || !winnerId || !scoreInput) {
            setError('Please fill in all fields')
            return
        }

        if (player1Id === player2Id) {
            setError('Player 1 and Player 2 must be different')
            return
        }

        if (winnerId !== player1Id && winnerId !== player2Id) {
            setError('Winner must be one of the selected players')
            return
        }

        try {
            setLoading(true)
            setError(null)

            const setScores = parseScores(scoreInput)

            await ladderService.addMatchResult(player1Id, player2Id, winnerId, setScores)

            // Reset form
            setPlayer1Id('')
            setPlayer2Id('')
            setWinnerId('')
            setScoreInput('')
            onMatchAdded()
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to add match')
        } finally {
            setLoading(false)
        }
    }

    const parseScores = (input: string): SetScore[] => {
        const sets = input.split(',').map(s => s.trim()).filter(s => s)
        const setScores: SetScore[] = []

        for (const set of sets) {
            const parts = set.split('-')
            if (parts.length !== 2) {
                throw new Error(`Invalid score format: "${set}". Use format like "11-9"`)
            }

            const p1ScoreStr = parts[0].trim()
            const p2ScoreStr = parts[1].trim()

            let p1Points = 0
            let p2Points = 0
            let p1Default = false
            let p2Default = false

            if (p1ScoreStr.toUpperCase() === 'D') {
                p1Default = true
            } else {
                p1Points = parseInt(p1ScoreStr)
                if (isNaN(p1Points)) throw new Error(`Invalid score number: "${p1ScoreStr}"`)
            }

            if (p2ScoreStr.toUpperCase() === 'D') {
                p2Default = true
            } else {
                p2Points = parseInt(p2ScoreStr)
                if (isNaN(p2Points)) throw new Error(`Invalid score number: "${p2ScoreStr}"`)
            }

            const setScore = new SetScore()
            setScore.setPlayer1Points(p1Points)
            setScore.setPlayer2Points(p2Points)
            setScore.setPlayer1Default(p1Default)
            setScore.setPlayer2Default(p2Default)

            setScores.push(setScore)
        }
        return setScores
    }

    return (
        <div className="add-match-form-container">
            <h3>Record Match Result</h3>
            <form onSubmit={handleSubmit} className="add-match-form">
                <div className="form-group">
                    <label>Player 1:</label>
                    <select value={player1Id} onChange={(e) => setPlayer1Id(e.target.value)} disabled={loading}>
                        <option value="">Select Player 1</option>
                        {players.map(p => (
                            <option key={p.getId()} value={p.getId()}>{p.getName()}</option>
                        ))}
                    </select>
                </div>

                <div className="form-group">
                    <label>Player 2:</label>
                    <select value={player2Id} onChange={(e) => setPlayer2Id(e.target.value)} disabled={loading}>
                        <option value="">Select Player 2</option>
                        {players.map(p => (
                            <option key={p.getId()} value={p.getId()}>{p.getName()}</option>
                        ))}
                    </select>
                </div>

                <div className="form-group">
                    <label>Winner:</label>
                    <div className="radio-group">
                        <label>
                            <input
                                type="radio"
                                name="winner"
                                value={player1Id}
                                checked={winnerId === player1Id && player1Id !== ''}
                                onChange={() => setWinnerId(player1Id)}
                                disabled={!player1Id || loading}
                            />
                            Player 1
                        </label>
                        <label>
                            <input
                                type="radio"
                                name="winner"
                                value={player2Id}
                                checked={winnerId === player2Id && player2Id !== ''}
                                onChange={() => setWinnerId(player2Id)}
                                disabled={!player2Id || loading}
                            />
                            Player 2
                        </label>
                    </div>
                </div>

                <div className="form-group">
                    <label>Score (e.g. "11-9, 5-11, 11-2"):</label>
                    <input
                        type="text"
                        value={scoreInput}
                        onChange={(e) => setScoreInput(e.target.value)}
                        placeholder="11-9, 6-11, 11-8"
                        disabled={loading}
                    />
                    <small>Use 'D' for default (e.g. "7-D")</small>
                </div>

                <button type="submit" disabled={loading} className="submit-match-btn">
                    {loading ? 'Recording...' : 'Record Match'}
                </button>
            </form>
            {error && <p className="error-message">{error}</p>}
        </div>
    )
}

export default AddMatchForm
