import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../AuthContext'
import { LadderServiceClient } from '../grpc/ladder_grpc_web_pb'
import { ClaimPlayerRequest } from '../grpc/ladder_pb'

// We need to inject the token into the gRPC client
const client = new LadderServiceClient(import.meta.env.VITE_API_URL || 'http://localhost:8080')

export default function Invite() {
    const { session, loading: authLoading } = useAuth()
    const { token } = useParams<{ token: string }>()
    const navigate = useNavigate()
    const [status, setStatus] = useState<'verifying' | 'claiming' | 'success' | 'error'>('verifying')
    const [errorMessage, setErrorMessage] = useState('')

    useEffect(() => {
        if (authLoading) return

        if (!session) {
            // If not logged in, we need them to login first.
            // We should probably redirect to login but preserve the invite URL to come back to?
            // For now, let's just show a message.
            setStatus('error')
            setErrorMessage('Please login to claim this invitation.')
            // Optionally: navigate('/login')
            return
        }

        if (!token) {
            setStatus('error')
            setErrorMessage('Invalid invitation link.')
            return
        }

        // Attempt to claim
        setStatus('claiming')
        const req = new ClaimPlayerRequest()
        req.setToken(token)

        // Add Auth Header
        const metadata = {
            'Authorization': `Bearer ${session.access_token}`
        }

        client.claimPlayer(req, metadata, (err, response) => {
            if (err) {
                setStatus('error')
                setErrorMessage(err.message)
            } else {
                if (response.getSuccess()) {
                    setStatus('success')
                    setTimeout(() => {
                        navigate('/')
                    }, 2000)
                } else {
                    setStatus('error')
                    setErrorMessage('Failed to claim player.')
                }
            }
        })

    }, [authLoading, session, token, navigate])

    if (authLoading || status === 'verifying') {
        return <div className="invite-container">Verifying invitation...</div>
    }

    if (status === 'error' && !session) {
        return (
            <div className="invite-container">
                <h1>Invitation</h1>
                <p>{errorMessage}</p>
                <button onClick={() => navigate('/login')}>Login to Claim</button>
            </div>
        )
    }

    return (
        <div className="invite-container">
            <div className="invite-box">
                {status === 'claiming' && <p>Claiming your profile...</p>}
                {status === 'success' && (
                    <>
                        <h1 style={{ color: 'green' }}>Success!</h1>
                        <p>You have successfully claimed your player profile.</p>
                        <p>Redirecting to dashboard...</p>
                    </>
                )}
                {status === 'error' && (
                    <>
                        <h1 style={{ color: 'red' }}>Error</h1>
                        <p>{errorMessage}</p>
                        <button onClick={() => navigate('/')}>Go Home</button>
                    </>
                )}
            </div>
            <style>{`
        .invite-container {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            height: 100vh;
            background: #f0f2f5;
        }
        .invite-box {
            background: white;
            padding: 2rem;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            text-align: center;
        }
        button {
            margin-top: 1rem;
            padding: 0.5rem 1rem;
            background: #007bff;
            color: white;
            border: none;
            cursor: pointer;
            border-radius: 4px;
        }
      `}</style>
        </div>
    )
}
