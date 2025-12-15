import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { isAdmin } from '../config';

export interface User {
    username: string;
    email: string | null;
    provider: string;
}

interface AuthContextType {
    user: User | null;
    login: (user: User) => void;
    logout: () => void;
    isLoading: boolean;
    isUserAdmin: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        const storedUser = localStorage.getItem('vstats_user');
        if (storedUser) {
            try {
                setUser(JSON.parse(storedUser));
            } catch (e) {
                console.error('Failed to parse stored user', e);
                localStorage.removeItem('vstats_user');
            }
        }
        setIsLoading(false);
    }, []);

    const login = (newUser: User) => {
        setUser(newUser);
        localStorage.setItem('vstats_user', JSON.stringify(newUser));
    };

    const logout = () => {
        setUser(null);
        localStorage.removeItem('vstats_user');
    };

    const isUserAdmin = isAdmin(user?.email);

    return (
        <AuthContext.Provider value={{ user, login, logout, isLoading, isUserAdmin }}>
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const context = useContext(AuthContext);
    if (context === undefined) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
}
