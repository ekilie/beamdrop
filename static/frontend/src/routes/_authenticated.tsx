import { createFileRoute, Outlet } from "@tanstack/react-router";
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/AppSidebar";
import { ThemeToggle } from "@/components/ThemeToggle";
import { Button } from "@/components/ui/button";
import { AuthProvider, useAuth } from "@/context/auth";
import Login from "@/pages/Login";
import { Menu, LogOut, Loader2 } from "lucide-react";

export const Route = createFileRoute("/_authenticated")({
    component: AuthenticatedLayout,
});

function AuthenticatedLayout() {
    return (
        <AuthProvider>
            <ProtectedRoute>
                <MainLayout />
            </ProtectedRoute>
        </AuthProvider>
    );
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated, isAuthEnabled, isLoading } = useAuth();

    if (isLoading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <div className="flex flex-col items-center gap-4 animate-fade-in">
                    <Loader2 className="w-12 h-12 animate-spin text-primary" />
                    <p className="text-muted-foreground font-mono uppercase text-sm">
                        Loading...
                    </p>
                </div>
            </div>
        );
    }

    if (isAuthEnabled && !isAuthenticated) {
        return <Login />;
    }

    return <>{children}</>;
}

function MainLayout() {
    const { isAuthEnabled, logout } = useAuth();

    return (
        <SidebarProvider>
            <div className="min-h-screen flex w-full">
                <AppSidebar password="" />
                <div className="flex-1 flex flex-col min-w-0">
                    <header className="h-14 flex items-center justify-between border-b border-border bg-card px-4 animate-fade-in shadow-sm">
                        <div className="flex items-center gap-3">
                            <SidebarTrigger className="p-2 hover:bg-muted rounded-md transition-smooth hover-lift lg:hidden">
                                <Menu className="w-5 h-5" />
                            </SidebarTrigger>
                            <div className="hidden sm:block">
                                <h1 className="text-lg font-bold font-mono uppercase tracking-wide text-foreground">
                                    BeamDrop
                                </h1>
                            </div>
                        </div>

                        <div className="flex items-center gap-2">
                            {isAuthEnabled && (
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={logout}
                                    className="gap-2 font-mono uppercase text-xs hover-lift transition-smooth"
                                >
                                    <LogOut className="w-4 h-4" />
                                    <span className="hidden sm:inline">Logout</span>
                                </Button>
                            )}
                            <ThemeToggle />
                        </div>
                    </header>

                    <main className="flex-1 overflow-y-auto scrollbar-thin">
                        <Outlet />
                    </main>
                </div>
            </div>
        </SidebarProvider>
    );
}
