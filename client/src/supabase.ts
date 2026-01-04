import { createClient } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY

if (!supabaseUrl || !supabaseAnonKey) {
    // Warn but don't crash, allowing dev to fix env
    console.error("Supabase URL or Key missing in VITE environment variables.")
}

export const supabase = createClient(supabaseUrl, supabaseAnonKey)
