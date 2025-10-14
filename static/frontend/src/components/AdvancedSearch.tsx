import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SearchIcon, Loader2 } from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { Card } from "@/components/ui/card";
import { getFileIcon } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";

interface SearchResult {
  name: string;
  size: string;
  isDir: boolean;
  modTime: string;
  path: string;
}

interface AdvancedSearchProps {
  currentPath: string;
  onNavigate: (path: string) => void;
  onPreview: (fileName: string) => void;
}

export function AdvancedSearch({ currentPath, onNavigate, onPreview }: AdvancedSearchProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [searchPath, setSearchPath] = useState(currentPath);
  const [isSearching, setIsSearching] = useState(false);
  const [results, setResults] = useState<SearchResult[]>([]);

  const handleSearch = async () => {
    if (!query.trim()) {
      toast({
        title: "Error",
        description: "Please enter a search query",
        variant: "destructive",
      });
      return;
    }

    setIsSearching(true);
    try {
      const params = new URLSearchParams({
        q: query,
        ...(searchPath && searchPath !== "." && { path: searchPath }),
      });

      const response = await fetch(`/search?${params.toString()}`);

      if (response.ok) {
        const data = await response.json();
        setResults(data.results || []);
        
        if (data.count === 0) {
          toast({
            title: "No Results",
            description: `No files found matching "${query}"`,
          });
        } else {
          toast({
            title: "Search Complete",
            description: `Found ${data.count} result${data.count !== 1 ? "s" : ""}`,
          });
        }
      } else {
        toast({
          title: "Error",
          description: "Failed to search files",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to search files",
        variant: "destructive",
      });
    } finally {
      setIsSearching(false);
    }
  };

  const handleResultClick = (result: SearchResult) => {
    if (result.isDir) {
      onNavigate(result.path);
      setOpen(false);
    } else {
      onPreview(result.name);
      setOpen(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2">
          <SearchIcon className="w-4 h-4" />
          <span className="font-mono text-xs font-bold uppercase tracking-wide hidden md:inline">
            Advanced Search
          </span>
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="font-mono uppercase tracking-wide">Advanced File Search</DialogTitle>
          <DialogDescription className="font-mono text-xs">
            Search for files and folders across your file system
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="query" className="font-mono text-xs uppercase">
              Search Query
            </Label>
            <Input
              id="query"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  handleSearch();
                }
              }}
              placeholder="e.g., report.pdf"
              className="font-mono"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="searchPath" className="font-mono text-xs uppercase">
              Search Path (Optional)
            </Label>
            <Input
              id="searchPath"
              value={searchPath}
              onChange={(e) => setSearchPath(e.target.value)}
              placeholder="Leave empty to search all files"
              className="font-mono"
            />
          </div>

          {/* Results */}
          {results.length > 0 && (
            <div className="mt-4 space-y-2">
              <div className="flex items-center justify-between">
                <Label className="font-mono text-xs uppercase">Search Results</Label>
                <Badge variant="outline">{results.length} found</Badge>
              </div>
              <div className="max-h-[300px] overflow-y-auto space-y-2 border border-border rounded-lg p-2">
                {results.map((result, idx) => (
                  <Card
                    key={idx}
                    className="p-3 hover:bg-muted/50 cursor-pointer transition-colors"
                    onClick={() => handleResultClick(result)}
                  >
                    <div className="flex items-center gap-3">
                      <div className="text-muted-foreground">
                        {getFileIcon(result.name, "w-5 h-5")}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="font-mono text-sm font-medium truncate">{result.name}</p>
                        <p className="font-mono text-xs text-muted-foreground truncate">
                          {result.path}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="font-mono text-xs text-muted-foreground">{result.size}</p>
                      </div>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Close
          </Button>
          <Button onClick={handleSearch} disabled={isSearching || !query.trim()}>
            {isSearching ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Searching...
              </>
            ) : (
              <>
                <SearchIcon className="w-4 h-4 mr-2" />
                Search
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
