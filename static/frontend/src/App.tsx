import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/AppSidebar";
import { ThemeProvider } from "@/components/ThemeProvider";
import { ThemeToggle } from "@/components/ThemeToggle";
import { SettingsDialog } from "@/components/SettingsDialog";
import { Button } from "@/components/ui/button";
import { AuthProvider, useAuth } from "@/context/auth";
import Index from "./pages/Index";
import Login from "./pages/Login";
import NotFound from "./pages/NotFound";
import { ApiKeysPage } from "./components/ApiKeysPage";
import { Menu, LogOut, Loader2 } from "lucide-react";

const queryClient = new QueryClient();

// Protected route wrapper
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isAuthEnabled, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-4 animate-fade-in">
          <Loader2 className="w-12 h-12 animate-spin text-primary" />
          <p className="text-muted-foreground font-mono uppercase text-sm">Loading...</p>
        </div>
      </div>
    );
  }

  // If auth is enabled and user is not authenticated, show login page
  if (isAuthEnabled && !isAuthenticated) {
    return <Login />;
  }

  return <>{children}</>;
}

// Main layout with sidebar and header
function MainLayout() {
  const { isAuthEnabled, logout } = useAuth();

  return (
    <SidebarProvider>
      <div className="min-h-screen flex w-full">
        <AppSidebar password="" />
        <div className="flex-1 flex flex-col min-w-0">
          {/* Global header with sidebar trigger and controls */}
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
              {/* Logout button (only shown when auth is enabled) */}
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
              <SettingsDialog />
            </div>
          </header>

          <main className="flex-1 overflow-y-auto scrollbar-thin">
            <Routes>
              <Route path="/" element={<Index />} />
              <Route path="/api-keys" element={<ApiKeysPage />} />
              <Route path="*" element={<NotFound />} />
            </Routes>
          </main>
        </div>
      </div>
    </SidebarProvider>
  );
}

const App = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="system" storageKey="beamdrop-ui-theme">
        <TooltipProvider>
          <Toaster />
          <Sonner />
          <BrowserRouter>
            <AuthProvider>
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            </AuthProvider>
          </BrowserRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
};

export default App;
