import { useState } from 'react'
import { supabase } from '../supabase'

export default function Login() {
    const [loading, setLoading] = useState(false)
    const [email, setEmail] = useState('')

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault()
        try {
            setLoading(true)
            const { error } = await supabase.auth.signInWithOtp({
                email,
                options: {
                    emailRedirectTo: window.location.origin
                }
            })
            if (error) throw error
            alert('Check your email for the login link!')
        } catch (error) {
            alert(error instanceof Error ? error.message : 'An error occurred')
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="login-container">
            <div className="login-box">
                <h1>Squash Ladder</h1>
                <p>Sign in via magic link with your email</p>
                <form onSubmit={handleLogin}>
                    <input
                        type="email"
                        placeholder="Your email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        required
                    />
                    <button type="submit" disabled={loading}>
                        {loading ? 'Sending...' : 'Send Magic Link'}
                    </button>
                </form>
            </div>
        </div>
    )
}
