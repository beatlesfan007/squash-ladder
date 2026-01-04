import React, { createContext, useContext, useEffect, useState } from 'react'
import { Session, User } from '@supabase/supabase-js'
import { supabase } from './supabase'

interface AuthContextType {
    session: Session | null
    user: User | null
    loading: boolean
    isAdmin: boolean
}

const AuthContext = createContext<AuthContextType>({ session: null, user: null, loading: true, isAdmin: false })

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [user, setUser] = useState<User | null>(null)
    const [session, setSession] = useState<Session | null>(null)
    const [loading, setLoading] = useState(true)
    const [isAdmin, setIsAdmin] = useState(false)

    const checkAdmin = (user: User | null) => {
        if (!user) return false
        const roles = user.app_metadata?.roles as string[] | undefined
        return roles?.includes('admin') ?? false
    }

    useEffect(() => {
        // Check active sessions
        supabase.auth.getSession().then(({ data: { session } }) => {
            setSession(session)
            const u = session?.user ?? null
            setUser(u)
            setIsAdmin(checkAdmin(u))
            setLoading(false)
        })

        // Listen for changes
        const { data: { subscription } } = supabase.auth.onAuthStateChange((_event, session) => {
            setSession(session)
            const u = session?.user ?? null
            setUser(u)
            setIsAdmin(checkAdmin(u))
            setLoading(false)
        })

        return () => subscription.unsubscribe()
    }, [])

    return (
        <AuthContext.Provider value={{ session, user, loading, isAdmin }}>
            {!loading && children}
        </AuthContext.Provider>
    )
}

export const useAuth = () => useContext(AuthContext)
