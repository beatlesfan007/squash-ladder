import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '../AuthContext'
import { ladderService } from '../grpc/ladderService'

export default function Invite() {
    const { token } = useParams<{ token: string }>()
    const { user, loading: authLoading } = useAuth()
    const navigate = useNavigate()
    const [claiming, setClaiming] = useState(false)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        if (!authLoading && !user) {
            // Store token in session storage to resume after login
            sessionStorage.setItem('pending_invite_token', token || '')
            navigate('/login')
        }
    }, [user, authLoading, token, navigate])

    const handleClaim = async () => {
        if (!token || !user) return
        try {
            setClaiming(true)
            const response = await ladderService.claimPlayer(token)
            if (response.getSuccess()) {
                alert('Successfully claimed player profile!')
                navigate('/')
            } else {
                setError('Failed to claim player.')
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'An error occurred during claiming')
        } finally {
            setClaiming(false)
        }
    }

    if (authLoading) return <div>Checking authentication...</div>

    return (
        <div className="invite-container">
            <h1>Claim Player Profile</h1>
            {error && <div className="error-banner">{error}</div>}
            <p>You have been invited to join the squash ladder. Click below to associate your account with your player profile.</p>
            <button onClick={handleClaim} disabled={claiming || !user}>
                {claiming ? 'Claiming...' : 'Claim My Profile'}
            </button>
        </div>
    )
}
