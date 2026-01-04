import { useState } from 'react'
import { supabase } from '../supabase'
// import { useNavigate } from 'react-router-dom'

export default function Login() {
  const [loading, setLoading] = useState(false)
  const [email, setEmail] = useState('')
  // const navigate = useNavigate()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    // Using Magic Link for simplicity and security
    const { error } = await supabase.auth.signInWithOtp({ email })

    if (error) {
      alert(error.message)
    } else {
      alert('Check your email for the login link!')
    }
    setLoading(false)
  }

  // Also support Google/GitHub if configured, but keeping it simple for now

  return (
    <div className="login-container theme-light">
      <div className="login-box">
        <h1 className="login-header">Squash Ladder Login</h1>
        <p className="login-description">Sign in via Magic Link</p>
        <form onSubmit={handleLogin} className="login-form">
          <input
            className="input-field"
            type="email"
            placeholder="Your email"
            value={email}
            required={true}
            onChange={(e) => setEmail(e.target.value)}
          />
          <button className={'button-primary'} disabled={loading}>
            {loading ? <span>Loading</span> : <span>Send Magic Link</span>}
          </button>
        </form>
      </div>
      <style>{`
        .login-container {
          display: flex;
          justify-content: center;
          align-items: center;
          height: 100vh;
          background-color: var(--background-color, #f0f2f5);
        }
        .login-box {
          background: white;
          padding: 2rem;
          border-radius: 8px;
          box-shadow: 0 4px 6px rgba(0,0,0,0.1);
          width: 100%;
          max-width: 400px;
          text-align: center;
        }
        .login-header {
          margin-bottom: 0.5rem;
          color: #333;
        }
        .login-description {
          margin-bottom: 2rem;
          color: #666;
        }
        .login-form {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }
        .input-field {
          padding: 0.75rem;
          border: 1px solid #ccc;
          border-radius: 4px;
          font-size: 1rem;
        }
        .button-primary {
          padding: 0.75rem;
          background-color: #007bff;
          color: white;
          border: none;
          border-radius: 4px;
          font-size: 1rem;
          cursor: pointer;
          transition: background-color 0.2s;
        }
        .button-primary:hover {
          background-color: #0056b3;
        }
        .button-primary:disabled {
          background-color: #ccc;
          cursor: not-allowed;
        }
      `}</style>
    </div>
  )
}
