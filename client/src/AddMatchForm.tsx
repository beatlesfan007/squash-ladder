import React, { useState } from 'react'
import { ladderService, SetScore, Player } from './grpc/ladderService'

interface AddMatchFormProps {
    players: Player[]
    onMatchAdded: () => void
}

const AddMatchForm: React.FC<AddMatchFormProps> = ({ players, onMatchAdded }) => {
    const [challengerId, setChallengerId] = useState('')
    const [defenderId, setDefenderId] = useState('')
    const [winnerId, setWinnerId] = useState('')
    const [scoreInput, setScoreInput] = useState('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!challengerId || !defenderId || !winnerId || !scoreInput) {
            setError('Please fill in all fields')
            return
        }

        if (challengerId === defenderId) {
            setError('Challenger and Defender must be different')
            return
        }

        if (winnerId !== challengerId && winnerId !== defenderId) {
            setError('Winner must be one of the selected players')
            return
        }

        try {
            setLoading(true)
            setError(null)

            const setScores = parseScores(scoreInput)

            await ladderService.addMatchResult(challengerId, defenderId, winnerId, setScores)

            // Reset form
            setChallengerId('')
            setDefenderId('')
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

            const challengerScoreStr = parts[0].trim()
            const defenderScoreStr = parts[1].trim()

            let challengerPoints = 0
            let defenderPoints = 0
            let challengerDefault = false
            let defenderDefault = false

            if (challengerScoreStr.toUpperCase() === 'D') {
                challengerDefault = true
            } else {
                challengerPoints = parseInt(challengerScoreStr)
                if (isNaN(challengerPoints)) throw new Error(`Invalid score number: "${challengerScoreStr}"`)
            }

            if (defenderScoreStr.toUpperCase() === 'D') {
                defenderDefault = true
            } else {
                defenderPoints = parseInt(defenderScoreStr)
                if (isNaN(defenderPoints)) throw new Error(`Invalid score number: "${defenderScoreStr}"`)
            }

            const setScore = new SetScore()
            setScore.setChallengerPoints(challengerPoints)
            setScore.setDefenderPoints(defenderPoints)
            setScore.setChallengerDefault(challengerDefault)
            setScore.setDefenderDefault(defenderDefault)

            setScores.push(setScore)
        }
        return setScores
    }

    return (
        <div className="add-match-form-container">
            <h3>Record Match Result</h3>
            <form onSubmit={handleSubmit} className="add-match-form">
                <div className="form-group">
                    <label>Challenger:</label>
                    <select value={challengerId} onChange={(e) => setChallengerId(e.target.value)} disabled={loading}>
                        <option value="">Select Challenger</option>
                        {players.map(p => (
                            <option key={p.getId()} value={p.getId()}>{p.getName()}</option>
                        ))}
                    </select>
                </div>

                <div className="form-group">
                    <label>Defender:</label>
                    <select value={defenderId} onChange={(e) => setDefenderId(e.target.value)} disabled={loading}>
                        <option value="">Select Defender</option>
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
                                value={challengerId}
                                checked={winnerId === challengerId && challengerId !== ''}
                                onChange={() => setWinnerId(challengerId)}
                                disabled={!challengerId || loading}
                            />
                            Challenger
                        </label>
                        <label>
                            <input
                                type="radio"
                                name="winner"
                                value={defenderId}
                                checked={winnerId === defenderId && defenderId !== ''}
                                onChange={() => setWinnerId(defenderId)}
                                disabled={!defenderId || loading}
                            />
                            Defender
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
