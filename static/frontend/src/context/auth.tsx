import React, { createContext, useContext, useState, useEffect, useCallback } from "react";

interface AuthContextType {
    isAuthenticated: boolean;
    isAuthEnabled: boolean;
    isLoading: boolean;
    login: (password: string) => Promise<boolean>;
    logout: () => Promise<void>;
    checkAuthStatus: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const [isAuthEnabled, setIsAuthEnabled] = useState(false);
    const [isLoading, setIsLoading] = useState(true);

    const checkAuthStatus = useCallback(async () => {
        try {
            const response = await fetch("/auth/status");
            if (response.ok) {
                const data = await response.json();
                setIsAuthEnabled(data.authEnabled);
                setIsAuthenticated(data.authenticated);
            }
        } catch (error) {
            console.error("Failed to check auth status:", error);
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        checkAuthStatus();
    }, [checkAuthStatus]);

    const login = async (password: string): Promise<boolean> => {
        try {
            const response = await fetch("/auth/login", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({ password }),
            });

            const data = await response.json();

            if (data.success) {
                // Store token in localStorage as backup
                if (data.token) {
                    localStorage.setItem("beamdrop_token", data.token);
                }
                setIsAuthenticated(true);
                return true;
            }

            return false;
        } catch (error) {
            console.error("Login failed:", error);
            return false;
        }
    };

    const logout = async (): Promise<void> => {
        try {
            await fetch("/auth/logout", {
                method: "POST",
            });
        } catch (error) {
            console.error("Logout failed:", error);
        } finally {
            localStorage.removeItem("beamdrop_token");
            setIsAuthenticated(false);
        }
    };

    return (
        <AuthContext.Provider
            value={{
                isAuthenticated,
                isAuthEnabled,
                isLoading,
                login,
                logout,
                checkAuthStatus,
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const context = useContext(AuthContext);
    if (context === undefined) {
        throw new Error("useAuth must be used within an AuthProvider");
    }
    return context;
}
