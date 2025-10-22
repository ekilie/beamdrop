import { useState } from "react";
import { Lock, Eye, EyeOff, Shield, KeyRound } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/hooks/use-toast";

interface PasswordDialogProps {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onPasswordSubmit?: (password: string) => void;
  triggerButton?: React.ReactNode;
}

export function PasswordDialog({
  open: controlledOpen,
  onOpenChange,
  onPasswordSubmit,
  triggerButton,
}: PasswordDialogProps) {
  const [internalOpen, setInternalOpen] = useState(false);
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [isValidating, setIsValidating] = useState(false);

  const open = controlledOpen ?? internalOpen;
  const setOpen = onOpenChange ?? setInternalOpen;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!password.trim()) {
      toast({
        title: "Password Required",
        description: "Please enter a password",
        variant: "destructive",
      });
      return;
    }

    setIsValidating(true);

    // Simulate password validation
    await new Promise(resolve => setTimeout(resolve, 800));

    if (onPasswordSubmit) {
      onPasswordSubmit(password);
    }

    toast({
      title: "Access Granted",
      description: "Password accepted successfully",
    });

    setIsValidating(false);
    setPassword("");
    setOpen(false);
  };

  const handleCancel = () => {
    setPassword("");
    setShowPassword(false);
    setOpen(false);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {triggerButton && (
        <DialogTrigger asChild>
          {triggerButton}
        </DialogTrigger>
      )}
      <DialogContent className="sm:max-w-[480px] border-2 border-border bg-gradient-to-br from-card via-card to-card/95 animate-fade-in">
        <DialogHeader className="space-y-4">
          <div className="mx-auto animate-scale-in">
            <div className="relative">
              <div className="absolute inset-0 bg-primary/20 blur-xl rounded-full animate-pulse" />
              <div className="relative bg-gradient-to-br from-primary to-primary/80 p-4 rounded-2xl border-2 border-primary/30 shadow-lg">
                <Shield className="w-8 h-8 text-primary-foreground animate-pulse" />
              </div>
            </div>
          </div>
          
          <DialogTitle className="text-2xl font-bold font-mono uppercase tracking-wide text-center text-foreground animate-fade-in">
            <span className="bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
              Secure Access
            </span>
          </DialogTitle>
          
          <DialogDescription className="text-center text-muted-foreground font-mono text-sm animate-fade-in">
            ENTER PASSWORD TO UNLOCK PROTECTED FEATURES
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6 mt-4 animate-slide-up">
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
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter your password..."
                className="pr-12 font-mono border-2 transition-all duration-300 focus:border-primary focus:shadow-lg focus:shadow-primary/20"
                disabled={isValidating}
              />
              
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8 p-0 hover:bg-muted transition-smooth"
                onClick={() => setShowPassword(!showPassword)}
                disabled={isValidating}
              >
                {showPassword ? (
                  <EyeOff className="w-4 h-4 text-muted-foreground" />
                ) : (
                  <Eye className="w-4 h-4 text-muted-foreground" />
                )}
              </Button>
            </div>

            {password.length > 0 && (
              <div className="flex items-center gap-2 animate-fade-in">
                <Badge 
                  variant={password.length >= 8 ? "default" : "outline"}
                  className="font-mono text-xs transition-smooth"
                >
                  {password.length >= 8 ? "STRONG" : "WEAK"}
                </Badge>
                <span className="text-xs font-mono text-muted-foreground">
                  {password.length} characters
                </span>
              </div>
            )}
          </div>

          <div className="flex flex-col sm:flex-row gap-3 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={handleCancel}
              disabled={isValidating}
              className="flex-1 font-mono uppercase transition-smooth hover:scale-105"
            >
              Cancel
            </Button>
            
            <Button
              type="submit"
              variant="utilitarian"
              disabled={isValidating || !password.trim()}
              className="flex-1 font-mono uppercase transition-smooth hover:scale-105 relative overflow-hidden group"
            >
              {isValidating ? (
                <>
                  <div className="absolute inset-0 bg-gradient-to-r from-primary/0 via-primary/20 to-primary/0 animate-shimmer" />
                  <span>VALIDATING...</span>
                </>
              ) : (
                <>
                  <Lock className="w-4 h-4 mr-2 group-hover:animate-pulse" />
                  AUTHENTICATE
                </>
              )}
            </Button>
          </div>
        </form>

        {/* Security indicator */}
        <div className="flex items-center justify-center gap-2 pt-4 border-t border-border/50 animate-fade-in">
          <div className="w-2 h-2 rounded-full bg-accent animate-pulse" />
          <span className="text-xs font-mono text-muted-foreground uppercase tracking-wide">
            End-to-End Protected
          </span>
        </div>
      </DialogContent>
    </Dialog>
  );
}
