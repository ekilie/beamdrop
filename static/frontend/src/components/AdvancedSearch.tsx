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
import {
  SearchIcon,
  Loader2,
  FolderOpen,
  File,
  Calendar,
  HardDrive,
} from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { Card } from "@/components/ui/card";
import { getFileIcon } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";

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

export function AdvancedSearch({
  currentPath,
  onNavigate,
  onPreview,
}: AdvancedSearchProps) {
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
    setResults([]);
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
            title: "No Results Found",
            description: `No files or folders matching "${query}"`,
          });
        } else {
          toast({
            title: "Search Complete",
            description: `Found ${data.count} result${
              data.count !== 1 ? "s" : ""
            }`,
          });
        }
      } else {
        throw new Error("Search failed");
      }
    } catch (error) {
      toast({
        title: "Search Error",
        description: "Failed to search files. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsSearching(false);
    }
  };

  const handleResultClick = (result: SearchResult) => {
    if (result.isDir) {
      onNavigate(result.path);
    } else {
      onPreview(result.name);
    }
    setOpen(false);
  };

  const formatFileSize = (size: string) => {
    if (!size || size === "0 B") return "-";
    return size;
  };

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleDateString();
    } catch {
      return "Unknown";
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
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] w-[95vw] flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <SearchIcon className="w-5 h-5" />
            <span className="font-mono uppercase tracking-wide">
              Advanced File Search
            </span>
          </DialogTitle>
          <DialogDescription className="font-mono text-xs">
            Search for files and folders across your file system with powerful
            filtering
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col flex-1 min-h-0 gap-4">
          <div className="grid gap-4 flex-shrink-0">
            <div className="grid gap-2">
              <Label
                htmlFor="query"
                className="font-mono text-xs uppercase flex items-center gap-2"
              >
                <SearchIcon className="w-3 h-3" />
                Search Query
              </Label>
              <Input
                id="query"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !isSearching) {
                    handleSearch();
                  }
                }}
                placeholder="Enter filename, extension, or pattern (e.g., *.pdf, report.*)"
                className="font-mono"
                disabled={isSearching}
              />
            </div>

            <div className="grid gap-2">
              <Label
                htmlFor="searchPath"
                className="font-mono text-xs uppercase flex items-center gap-2"
              >
                <FolderOpen className="w-3 h-3" />
                Search Path (Optional)
              </Label>
              <Input
                id="searchPath"
                value={searchPath}
                onChange={(e) => setSearchPath(e.target.value)}
                placeholder="Leave empty to search all files"
                className="font-mono"
                disabled={isSearching}
              />
              <p className="text-xs text-muted-foreground font-mono">
                Current path: {currentPath}
              </p>
            </div>
          </div>

          <Separator className="flex-shrink-0" />

          <div className="flex-1 min-h-0 overflow-auto">
            {/* Results Section */}
            {results.length > 0 && (
              <div className="h-full flex flex-col">
                <div className="flex items-center justify-between mb-3 flex-shrink-0">
                  <Label className="font-mono text-xs uppercase">
                    Search Results
                  </Label>
                  <Badge variant="secondary" className="font-mono">
                    {results.length} {results.length === 1 ? "item" : "items"}
                  </Badge>
                </div>

                <ScrollArea className="flex-1 border rounded-lg">
                  <div className="p-2 space-y-2">
                    {results.map((result, idx) => (
                      <Card
                        key={`${result.path}-${idx}`}
                        className="p-3 hover:bg-muted/50 cursor-pointer transition-colors border-border/50"
                        onClick={() => handleResultClick(result)}
                      >
                        <div className="flex items-start gap-3">
                          <div className="flex-shrink-0 mt-0.5 text-muted-foreground">
                            {result.isDir ? (
                              <FolderOpen className="w-5 h-5 text-blue-500" />
                            ) : (
                              getFileIcon(result.name, "w-5 h-5") || (
                                <File className="w-5 h-5" />
                              )
                            )}
                          </div>

                          <div className="flex-1 min-w-0 space-y-1">
                            <div className="flex items-center gap-2">
                              <p
                                className="font-medium text-sm truncate"
                                title={result.name}
                              >
                                {result.name}
                              </p>
                              {result.isDir && (
                                <Badge variant="outline" className="text-xs">
                                  Folder
                                </Badge>
                              )}
                            </div>

                            <p
                              className="text-xs text-muted-foreground truncate"
                              title={result.path}
                            >
                              {result.path}
                            </p>

                            <div className="flex items-center gap-4 text-xs text-muted-foreground">
                              <div className="flex items-center gap-1">
                                <HardDrive className="w-3 h-3" />
                                <span>{formatFileSize(result.size)}</span>
                              </div>
                              <div className="flex items-center gap-1">
                                <Calendar className="w-3 h-3" />
                                <span>{formatDate(result.modTime)}</span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </Card>
                    ))}
                  </div>
                </ScrollArea>
              </div>
            )}

            {/* Empty State */}
            {!isSearching && results.length === 0 && query && (
              <div className="h-full flex items-center justify-center text-center p-8">
                <div className="space-y-2">
                  <SearchIcon className="w-12 h-12 text-muted-foreground/50 mx-auto" />
                  <p className="font-mono text-sm text-muted-foreground">
                    No results found for "{query}"
                  </p>
                  <p className="font-mono text-xs text-muted-foreground">
                    Try adjusting your search terms or path
                  </p>
                </div>
              </div>
            )}

            {/* Loading State */}
            {isSearching && (
              <div className="h-full flex items-center justify-center text-center p-8">
                <div className="space-y-3">
                  <Loader2 className="w-8 h-8 animate-spin text-primary mx-auto" />
                  <p className="font-mono text-sm text-muted-foreground">
                    Searching for "{query}"...
                  </p>
                </div>
              </div>
            )}

            {/* Initial State - No search performed */}
            {!isSearching && results.length === 0 && !query && (
              <div className="h-full flex items-center justify-center text-center p-8">
                <div className="space-y-2">
                  <SearchIcon className="w-12 h-12 text-muted-foreground/50 mx-auto" />
                  <p className="font-mono text-sm text-muted-foreground">
                    Enter a search query to find files and folders
                  </p>
                  <p className="font-mono text-xs text-muted-foreground">
                    You can search by name, extension, or pattern
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="flex-shrink-0 gap-2 sm:gap-0 mt-2">
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={isSearching}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSearch}
            disabled={isSearching || !query.trim()}
            className="min-w-24"
          >
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
