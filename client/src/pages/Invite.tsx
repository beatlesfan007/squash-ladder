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
            <div className="invite-box">
                <h1 className="invite-header">Claim Player Profile</h1>
                {error && <div className="error-banner">{error}</div>}
                <p className="invite-text">You have been invited to join the squash ladder. Click below to associate your account with your player profile.</p>
                <button
                    onClick={handleClaim}
                    disabled={claiming || !user}
                    className="button-primary"
                >
                    {claiming ? 'Claiming...' : 'Claim My Profile'}
                </button>
            </div>
            <style>{`
                .invite-container {
                    min-height: 100vh;
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    justify-content: center;
                    background-color: #f0f2f5;
                    padding: 1rem;
                }
                .invite-box {
                    background: white;
                    padding: 2.5rem;
                    border-radius: 12px;
                    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
                    width: 100%;
                    max-width: 400px;
                    text-align: center;
                }
                .invite-header {
                    margin: 0 0 1rem 0;
                    color: #1a1a1a;
                    font-size: 2rem;
                }
                .invite-text {
                    color: #666;
                    margin-bottom: 2rem;
                    line-height: 1.5;
                }
                .button-primary {
                    width: 100%;
                    padding: 0.8rem;
                    font-size: 1.1rem;
                    border-radius: 8px;
                    background-color: #2196f3;
                    color: white;
                    border: none;
                    cursor: pointer;
                    transition: background-color 0.2s, transform 0.1s;
                }
                .button-primary:hover {
                    background-color: #1976d2;
                }
                .button-primary:active {
                    transform: translateY(1px);
                }
                .button-primary:disabled {
                    background-color: #ccc;
                    cursor: not-allowed;
                    opacity: 0.7;
                }
                .error-banner {
                    background-color: #ffebee;
                    color: #c62828;
                    padding: 1rem;
                    margin-bottom: 1.5rem;
                    border-radius: 4px;
                    border: 1px solid #ffcdd2;
                    font-size: 0.9rem;
                }
            `}</style>
        </div>
    )
}
