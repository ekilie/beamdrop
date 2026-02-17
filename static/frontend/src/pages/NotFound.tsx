import { useRouter, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { FileQuestion, ArrowLeft } from "lucide-react";

const NotFound = () => {
  const router = useRouter();
  const navigate = useNavigate();
  const pathname = router.state.location.pathname;

  useEffect(() => {
    console.error(
      "404 Error: User attempted to access non-existent route:",
      pathname
    );
  }, [pathname]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background via-background to-muted/20 p-4">
      {/* Background decoration */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-64 h-64 bg-primary/5 rounded-full blur-3xl animate-pulse" />
        <div className="absolute bottom-1/4 right-1/4 w-48 h-48 bg-accent/5 rounded-full blur-3xl animate-pulse" />
      </div>

      <div className="relative w-full max-w-md animate-fade-in">
        <div className="bg-card border-2 border-border rounded-2xl p-8 shadow-2xl animate-scale-in text-center">
          <div className="flex justify-center mb-6">
            <div className="p-4 rounded-2xl bg-muted/50 border border-border">
              <FileQuestion className="w-12 h-12 text-muted-foreground" />
            </div>
          </div>

          <h1 className="text-6xl font-bold font-mono uppercase tracking-wider bg-gradient-to-r from-primary via-accent to-primary bg-clip-text text-transparent mb-2 animate-slide-up">
            404
          </h1>

          <p className="text-lg font-mono uppercase tracking-wide text-foreground mb-2">
            Page not found
          </p>

          <p className="text-sm font-mono text-muted-foreground uppercase tracking-wide mb-6">
            The route{" "}
            <span className="text-primary break-all normal-case">{pathname}</span>{" "}
            doesn't exist
          </p>

          <Button
            onClick={() => navigate({ to: "/" })}
            className="w-full font-mono uppercase tracking-wide transition-smooth hover:scale-[1.02] gap-2"
          >
            <ArrowLeft className="w-4 h-4" />
            Return to Home
          </Button>
        </div>

        <p className="text-center mt-4 text-xs font-mono text-muted-foreground uppercase tracking-wide animate-fade-in">
          BeamDrop
        </p>
      </div>
    </div>
  );
};

export default NotFound;
