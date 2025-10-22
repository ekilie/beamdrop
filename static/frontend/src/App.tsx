import { useState } from "react";
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
import { PasswordDialog } from "@/components/PasswordDialog";
import { Button } from "@/components/ui/button";
import Index from "./pages/Index";
import NotFound from "./pages/NotFound";
import { Menu, Lock } from "lucide-react";

const queryClient = new QueryClient();

const App = () => {
  const [password, setPassword] = useState("");

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="system" storageKey="beamdrop-ui-theme">
        <TooltipProvider>
          <Toaster />
          <Sonner />
          <BrowserRouter>
            <SidebarProvider>
              <div className="min-h-screen flex w-full">
                <AppSidebar password={password} />
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
                      {/* Password Test Button */}
                      <PasswordDialog
                        onPasswordSubmit={setPassword}
                        triggerButton={
                          <Button 
                            variant="outline" 
                            size="sm"
                            className="gap-2 font-mono uppercase text-xs hover-lift transition-smooth"
                          >
                            <Lock className="w-4 h-4" />
                            <span className="hidden sm:inline">Auth</span>
                          </Button>
                        }
                      />
                      <ThemeToggle />
                      <SettingsDialog />
                    </div>
                  </header>
                  
                  <main className="flex-1 overflow-y-auto scrollbar-thin">
                    <Routes>
                      <Route path="/" element={<Index />} />
                      <Route path="*" element={<NotFound />} />
                    </Routes>
                  </main>
                </div>
              </div>
            </SidebarProvider>
          </BrowserRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
};

export default App;
