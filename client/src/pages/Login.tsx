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
                <h1 className="login-header">Squash Ladder</h1>
                <p className="login-description">Sign in via Magic Link with your email</p>
                <form onSubmit={handleLogin} className="login-form">
                    <input
                        className="input-field"
                        type="email"
                        placeholder="Your email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        required
                    />
                    <button className="button-primary" type="submit" disabled={loading}>
                        {loading ? 'Sending...' : 'Send Magic Link'}
                    </button>
                </form>
            </div>
            <style>{`
                .login-container {
                    min-height: 100vh;
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    justify-content: center;
                    background-color: #f0f2f5;
                    padding: 1rem;
                }
                .login-box {
                    background: white;
                    padding: 2.5rem;
                    border-radius: 12px;
                    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
                    width: 100%;
                    max-width: 400px;
                    text-align: center;
                }
                .login-header {
                    margin: 0 0 1rem 0;
                    color: #1a1a1a;
                    font-size: 2rem;
                }
                .login-description {
                    color: #666;
                    margin-bottom: 2rem;
                    line-height: 1.5;
                }
                .login-form {
                    display: flex;
                    flex-direction: column;
                    gap: 1rem;
                }
                .input-field {
                    padding: 0.8rem 1rem;
                    border: 1px solid #ddd;
                    border-radius: 8px;
                    font-size: 1rem;
                    transition: border-color 0.2s;
                    width: 100%;
                    box-sizing: border-box;
                }
                .input-field:focus {
                    border-color: #2196f3;
                    outline: none;
                    box-shadow: 0 0 0 3px rgba(33, 150, 243, 0.1);
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
                }
            `}</style>
        </div>
    )
}
