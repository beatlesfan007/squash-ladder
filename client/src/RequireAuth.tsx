import { Navigate } from 'react-router-dom'
import { useAuth } from './AuthContext'

export default function RequireAuth({ children }: { children: JSX.Element }) {
    const { user, loading } = useAuth()

    if (loading) {
        return <div>Loading authentication...</div>
    }

    if (!user) {
        return <Navigate to="/login" replace />
    }

    // Check for pending invite redirect
    const pendingToken = localStorage.getItem('pending_invite_token')
    if (pendingToken) {
        localStorage.removeItem('pending_invite_token')
        return <Navigate to={`/invite/${pendingToken}`} replace />
    }

    return children
}
