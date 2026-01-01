import React, { useState } from 'react'
import { ladderService } from './grpc/ladderService'

interface AddPlayerFormProps {
    onPlayerAdded: () => void
}

const AddPlayerForm: React.FC<AddPlayerFormProps> = ({ onPlayerAdded }) => {
    const [name, setName] = useState('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!name.trim()) return

        try {
            setLoading(true)
            setError(null)
            await ladderService.addPlayer(name)
            setName('')
            onPlayerAdded()
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to add player')
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="add-player-form-container">
            <h3>Add New Player</h3>
            <form onSubmit={handleSubmit} className="add-player-form">
                <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Player Name"
                    disabled={loading}
                    className="player-name-input"
                />
                <button type="submit" disabled={loading || !name.trim()} className="add-player-btn">
                    {loading ? 'Adding...' : 'Add Player'}
                </button>
            </form>
            {error && <p className="error-message">{error}</p>}
        </div>
    )
}

export default AddPlayerForm
