import React, { useState } from 'react'
import { ladderService } from './grpc/ladderService'

interface AddPlayerFormProps {
    onPlayerAdded: () => void
}

const AddPlayerForm: React.FC<AddPlayerFormProps> = ({ onPlayerAdded }) => {
    const [name, setName] = useState('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const [inviteLink, setInviteLink] = useState<string | null>(null)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!name.trim()) return

        try {
            setLoading(true)
            setError(null)
            setInviteLink(null)
            const player = await ladderService.addPlayer(name)
            setName('')
            onPlayerAdded()

            try {
                const token = await ladderService.generateInvite(player.getId())
                const link = `${window.location.origin}/invite/${token}`
                setInviteLink(link)
            } catch (inviteErr) {
                console.error("Failed to generate invite automatically", inviteErr)
            }

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
            {inviteLink && (
                <div className="invite-result">
                    <p>Player added! Send this invite link:</p>
                    <div className="invite-link-box">
                        <input readOnly value={inviteLink} onClick={e => e.currentTarget.select()} />
                        <button onClick={() => navigator.clipboard.writeText(inviteLink)}>Copy</button>
                    </div>
                </div>
            )}
            <style>{`
                .invite-result {
                    margin-top: 1rem;
                    padding: 1rem;
                    background: #e6fffa;
                    border: 1px solid #38b2ac;
                    border-radius: 4px;
                }
                .invite-link-box {
                    display: flex;
                    gap: 0.5rem;
                    margin-top: 0.5rem;
                }
                .invite-link-box input {
                    flex: 1;
                    padding: 0.5rem;
                }
            `}</style>
        </div>
    )
}

export default AddPlayerForm
