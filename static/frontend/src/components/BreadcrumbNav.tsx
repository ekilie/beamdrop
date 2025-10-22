import React from "react";
import { Button } from "@/components/ui/button";
import { ChevronRight, Home, Folder } from "lucide-react";
import { cn } from "@/lib/utils";

interface BreadcrumbNavProps {
  currentPath: string;
  onNavigate: (path: string) => void;
  className?: string;
}

export function BreadcrumbNav({ currentPath, onNavigate, className }: BreadcrumbNavProps) {
  const pathParts = currentPath === "." ? [] : currentPath.split("/").filter(Boolean);

  const handleNavigate = (index: number) => {
    if (index === -1) {
      onNavigate(".");
    } else {
      const newPath = pathParts.slice(0, index + 1).join("/");
      onNavigate(newPath);
    }
  };

  return (
    <div className={cn("overflow-x-auto scrollbar-thin scrollbar-thumb-muted scrollbar-track-transparent animate-fade-in", className)}>
      <nav className="flex items-center gap-1 text-sm min-w-max py-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => handleNavigate(-1)}
          className="flex items-center gap-2 px-3 py-2 h-auto font-mono text-xs hover:bg-muted flex-shrink-0 transition-smooth hover-lift rounded-md"
        >
          <Home className="w-4 h-4" />
          <span className="uppercase font-bold tracking-wide hidden sm:inline">ROOT</span>
        </Button>

        {pathParts.map((part, index) => (
          <React.Fragment key={index}>
            <ChevronRight className="w-4 h-4 text-muted-foreground flex-shrink-0 animate-fade-in" />
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleNavigate(index)}
              className="flex items-center gap-2 px-3 py-2 h-auto font-mono text-xs hover:bg-muted flex-shrink-0 transition-smooth hover-lift rounded-md animate-fade-in"
              style={{ animationDelay: `${index * 50}ms` }}
              title={part}
            >
              <Folder className="w-4 h-4" />
              <span className="uppercase font-bold tracking-wide max-w-[100px] sm:max-w-[150px] truncate">
                {part}
              </span>
            </Button>
          </React.Fragment>
        ))}
      </nav>
    </div>
  );
}