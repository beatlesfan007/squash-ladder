import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from './AuthContext'

export default function RequireAuth({ children }: { children: JSX.Element }) {
    const { session, loading } = useAuth()
    const location = useLocation()

    if (loading) {
        // Or a spinner
        return <div>Loading...</div>
    }

    if (!session) {
        // Redirect them to the /login page, but save the current location they were
        // trying to go to when they were redirected. This allows us to send them
        // along to that page after they login, which is a nicer user experience
        // than dropping them off on the home page.
        return <Navigate to="/login" state={{ from: location }} replace />
    }

    return children
}
