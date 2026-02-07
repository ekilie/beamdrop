import { useState } from "react";
import { Lock, Eye, EyeOff, Shield, KeyRound, Zap, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/context/auth";
import { toast } from "@/hooks/use-toast";

const Login = () => {
    const [password, setPassword] = useState("");
    const [showPassword, setShowPassword] = useState(false);
    const [isValidating, setIsValidating] = useState(false);
    const [error, setError] = useState("");
    const { login } = useAuth();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError("");

        if (!password.trim()) {
            setError("Password is required");
            return;
        }

        setIsValidating(true);

        try {
            const success = await login(password);

            if (success) {
                toast({
                    title: "Access Granted",
                    description: "Welcome to BeamDrop",
                });
            } else {
                setError("Invalid password. Please try again.");
                toast({
                    title: "Access Denied",
                    description: "Invalid password",
                    variant: "destructive",
                });
            }
        } catch (err) {
            setError("Authentication failed. Please try again.");
            toast({
                title: "Error",
                description: "Authentication failed",
                variant: "destructive",
            });
        } finally {
            setIsValidating(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background via-background to-muted/20 p-4">
            {/* Background decoration */}
            <div className="absolute inset-0 overflow-hidden pointer-events-none">
                <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-primary/5 rounded-full blur-3xl animate-pulse" />
                <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-accent/5 rounded-full blur-3xl animate-pulse delay-1000" />
            </div>

            <div className="relative w-full max-w-md animate-fade-in">
                {/* Logo and branding */}
                <div className="text-center mb-8 animate-slide-up">
                    {/* <div className="inline-flex items-center justify-center mb-4">
                        <div className="relative">
                            <div className="absolute inset-0 bg-primary/20 blur-2xl rounded-full animate-pulse" />
                            <div className="relative bg-gradient-to-br from-primary to-primary/80 p-5 rounded-3xl border-2 border-primary/30 shadow-2xl">
                                <Zap className="w-12 h-12 text-primary-foreground" />
                            </div>
                        </div>
                    </div> */}
                    <h1 className="text-4xl font-bold font-mono uppercase tracking-wider text-foreground mb-2">
                        <span className="bg-gradient-to-r from-primary via-accent to-primary bg-clip-text text-transparent">
                            BeamDrop
                        </span>
                    </h1>
                    <p className="text-muted-foreground font-mono text-sm uppercase tracking-wide">
                        Secure File Transfer
                    </p>
                </div>

                {/* Login card */}
                <div className="bg-card border-2 border-border rounded-2xl p-8 shadow-2xl animate-scale-in">
                    {/* Header */}
                    <div className="text-center mb-8">
                        <div className="mx-auto mb-4 animate-scale-in">
                            <div className="relative inline-block">
                                <div className="absolute inset-0 bg-primary/20 blur-xl rounded-full animate-pulse" />
                                <div className="relative bg-gradient-to-br from-primary to-primary/80 p-4 rounded-2xl border-2 border-primary/30 shadow-lg">
                                    <Shield className="w-8 h-8 text-primary-foreground animate-pulse" />
                                </div>
                            </div>
                        </div>

                        <h2 className="text-2xl font-bold font-mono uppercase tracking-wide text-foreground">
                            <span className="bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                                Secure Access
                            </span>
                        </h2>

                        <p className="text-muted-foreground font-mono text-sm mt-2 uppercase tracking-wide">
                            Enter password to continue
                        </p>
                    </div>

                    {/* Login form */}
                    <form onSubmit={handleSubmit} className="space-y-6">
                        <div className="space-y-3">
                            <Label
                                htmlFor="password"
                                className="text-sm font-mono uppercase tracking-wide text-foreground flex items-center gap-2"
                            >
                                <KeyRound className="w-4 h-4 text-primary" />
                                Password
                            </Label>

                            <div className="relative group">
                                <Input
                                    id="password"
                                    type={showPassword ? "text" : "password"}
                                    value={password}
                                    onChange={(e) => {
                                        setPassword(e.target.value);
                                        setError("");
                                    }}
                                    placeholder="Enter your password..."
                                    className="pr-12 font-mono border-2 transition-all duration-300 focus:border-primary focus:shadow-lg focus:shadow-primary/20 h-12 text-base"
                                    disabled={isValidating}
                                    autoFocus
                                />

                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    className="absolute right-1 top-1/2 -translate-y-1/2 h-10 w-10 p-0 hover:bg-muted transition-smooth"
                                    onClick={() => setShowPassword(!showPassword)}
                                    disabled={isValidating}
                                >
                                    {showPassword ? (
                                        <EyeOff className="w-5 h-5 text-muted-foreground" />
                                    ) : (
                                        <Eye className="w-5 h-5 text-muted-foreground" />
                                    )}
                                </Button>
                            </div>

                            {/* Error message */}
                            {error && (
                                <div className="flex items-center gap-2 text-destructive animate-fade-in">
                                    <AlertCircle className="w-4 h-4" />
                                    <span className="text-sm font-mono">{error}</span>
                                </div>
                            )}

                            
                        </div>

                        <Button
                            type="submit"
                            disabled={isValidating || !password.trim()}
                            className="w-full h-12 font-mono uppercase text-base transition-smooth hover:scale-[1.02] relative overflow-hidden group bg-primary hover:bg-primary/90"
                        >
                            {isValidating ? (
                                <>
                                    <div className="absolute inset-0 bg-gradient-to-r from-primary/0 via-white/20 to-primary/0 animate-shimmer" />
                                    <span className="flex items-center gap-2">
                                        <div className="w-5 h-5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
                                        Validating...
                                    </span>
                                </>
                            ) : (
                                <>
                                    <Lock className="w-5 h-5 mr-2 group-hover:animate-pulse" />
                                    Authenticate
                                </>
                            )}
                        </Button>
                    </form>

                    {/* Security indicator */}
                    <div className="flex items-center justify-center gap-2 pt-6 mt-6 border-t border-border/50 animate-fade-in">
                        <div className="w-2 h-2 rounded-full bg-accent animate-pulse" />
                        <span className="text-xs font-mono text-muted-foreground uppercase tracking-wide">
                            End-to-End Protected
                        </span>
                    </div>
                </div>

                {/* Footer */}
                <div className="text-center mt-6 animate-fade-in">
                    <p className="text-xs font-mono text-muted-foreground uppercase tracking-wide">
                        Protected by BeamDrop Security
                    </p>
                </div>
            </div>
        </div>
    );
};

export default Login;
