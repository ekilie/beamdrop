import { useState, useEffect, useCallback, useRef } from "react";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  Search,
  RefreshCw,
  FileText,
  AlertCircle,
  AlertTriangle,
  Info,
  Bug,
  ChevronDown,
  ChevronRight,
  Loader2,
  Download,
} from "lucide-react";
import Footer from "@/components/Footer";

interface LogEntry {
  time: string;
  level: string;
  msg: string;
  source?: {
    function?: string;
    file?: string;
    line?: number;
  };
  [key: string]: unknown;
}

interface LogsResponse {
  logs: LogEntry[];
  total: number;
  returned: number;
  hasMore: boolean;
  logPath: string;
}

const LEVEL_CONFIG: Record<
  string,
  { icon: React.ReactNode; color: string; bg: string; border: string }
> = {
  DEBUG: {
    icon: <Bug className="w-3.5 h-3.5" />,
    color: "text-cyan-500",
    bg: "bg-cyan-500/10",
    border: "border-cyan-500/30",
  },
  INFO: {
    icon: <Info className="w-3.5 h-3.5" />,
    color: "text-green-500",
    bg: "bg-green-500/10",
    border: "border-green-500/30",
  },
  WARN: {
    icon: <AlertTriangle className="w-3.5 h-3.5" />,
    color: "text-yellow-500",
    bg: "bg-yellow-500/10",
    border: "border-yellow-500/30",
  },
  ERROR: {
    icon: <AlertCircle className="w-3.5 h-3.5" />,
    color: "text-red-500",
    bg: "bg-red-500/10",
    border: "border-red-500/30",
  },
};

function getLevelConfig(level: string) {
  return (
    LEVEL_CONFIG[level.toUpperCase()] ?? {
      icon: <Info className="w-3.5 h-3.5" />,
      color: "text-muted-foreground",
      bg: "bg-muted/10",
      border: "border-muted",
    }
  );
}

function formatTime(iso: string) {
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      fractionalSecondDigits: 3,
    });
  } catch {
    return iso;
  }
}

function formatDate(iso: string) {
  try {
    const d = new Date(iso);
    return d.toLocaleDateString([], {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch {
    return "";
  }
}

/** Extract extra attributes (everything except time, level, msg, source). */
function getExtraAttrs(entry: LogEntry): Record<string, unknown> {
  const skip = new Set(["time", "level", "msg", "source"]);
  const extras: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(entry)) {
    if (!skip.has(k)) extras[k] = v;
  }
  return extras;
}

function LogRow({ entry }: { entry: LogEntry }) {
  const [expanded, setExpanded] = useState(false);
  const cfg = getLevelConfig(entry.level);
  const extras = getExtraAttrs(entry);
  const hasExtras =
    Object.keys(extras).length > 0 || entry.source?.file;

  return (
    <div className="border-b border-border/50 last:border-b-0">
      <button
        className="w-full text-left px-4 py-2.5 hover:bg-muted/30 transition-colors flex items-start gap-3 group"
        onClick={() => hasExtras && setExpanded(!expanded)}
        disabled={!hasExtras}
      >
        {/* Expand icon */}
        <div className="mt-0.5 text-muted-foreground/60 w-4 flex-shrink-0">
          {hasExtras &&
            (expanded ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronRight className="w-4 h-4" />
            ))}
        </div>

        {/* Timestamp */}
        <span className="text-xs font-mono text-muted-foreground whitespace-nowrap w-24 flex-shrink-0">
          {formatTime(entry.time)}
        </span>

        {/* Level badge */}
        <Badge
          variant="outline"
          className={`${cfg.bg} ${cfg.color} ${cfg.border} font-mono text-[10px] uppercase w-16 justify-center flex-shrink-0`}
        >
          <span className="flex items-center gap-1">
            {cfg.icon}
            {entry.level}
          </span>
        </Badge>

        {/* Message */}
        <span className="text-sm font-mono text-foreground truncate flex-1 min-w-0">
          {entry.msg}
        </span>
      </button>

      {/* Expanded details */}
      {expanded && (
        <div className="px-4 pb-3 pl-14 space-y-2 animate-fade-in">
          {/* Source info */}
          {entry.source?.file && (
            <div className="flex items-center gap-2 text-xs font-mono text-muted-foreground">
              <span className="text-muted-foreground/60">source:</span>
              <span>
                {entry.source.file}:{entry.source.line}
              </span>
              {entry.source.function && (
                <span className="text-muted-foreground/60">
                  ({entry.source.function})
                </span>
              )}
            </div>
          )}

          {/* Extra attributes */}
          {Object.keys(extras).length > 0 && (
            <div className="bg-muted/20 rounded-md p-3 border border-border/50">
              <pre className="text-xs font-mono text-foreground/80 whitespace-pre-wrap break-all">
                {JSON.stringify(extras, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function LogsPage() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [levelFilter, setLevelFilter] = useState<string>("all");
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [limit] = useState(200);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchLogs = useCallback(async () => {
    try {
      setIsLoading(true);
      const params = new URLSearchParams();
      params.set("limit", String(limit));
      if (levelFilter !== "all") params.set("level", levelFilter);
      if (searchTerm.trim()) params.set("search", searchTerm.trim());

      const res = await fetch(`/logs?${params.toString()}`);
      if (!res.ok) throw new Error("Failed to fetch logs");
      const data: LogsResponse = await res.json();

      setLogs(data.logs ?? []);
      setTotal(data.total);
      setHasMore(data.hasMore);
    } catch (err) {
      console.error("Failed to fetch logs", err);
    } finally {
      setIsLoading(false);
    }
  }, [limit, levelFilter, searchTerm]);

  // Initial fetch + search/filter changes
  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  // Auto-refresh
  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchLogs, 3000);
    } else if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, fetchLogs]);

  // Count by level
  const levelCounts = logs.reduce(
    (acc, log) => {
      const lvl = log.level?.toUpperCase() ?? "INFO";
      acc[lvl] = (acc[lvl] ?? 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const handleExport = () => {
    const blob = new Blob([JSON.stringify(logs, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `beamdrop-logs-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="bg-background min-h-screen flex flex-col">
      {/* Header */}
      <div className="border-b border-border bg-card px-4 sm:px-6 py-4">
        <div className="container mx-auto flex flex-col lg:flex-row lg:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="bg-primary p-2 rounded border border-primary">
              <FileText className="w-5 h-5 text-primary-foreground" />
            </div>
            <div>
              <h2 className="text-xl font-bold font-mono uppercase tracking-wide text-foreground">
                Server Logs
              </h2>
              <p className="text-muted-foreground font-mono text-xs">
                {total.toLocaleString()} TOTAL ENTRIES
              </p>
            </div>
          </div>

          {/* Controls */}
          <div className="flex flex-wrap items-center gap-3">
            {/* Search */}
            <div className="relative w-64">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder="SEARCH LOGS..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-10 font-mono text-sm uppercase tracking-wide border-2 border-border"
              />
            </div>

            {/* Level filter */}
            <Select value={levelFilter} onValueChange={setLevelFilter}>
              <SelectTrigger className="w-32 font-mono text-xs uppercase border-2 border-border">
                <SelectValue placeholder="ALL LEVELS" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all" className="font-mono text-xs">
                  ALL LEVELS
                </SelectItem>
                <SelectItem value="debug" className="font-mono text-xs">
                  DEBUG
                </SelectItem>
                <SelectItem value="info" className="font-mono text-xs">
                  INFO
                </SelectItem>
                <SelectItem value="warn" className="font-mono text-xs">
                  WARN
                </SelectItem>
                <SelectItem value="error" className="font-mono text-xs">
                  ERROR
                </SelectItem>
              </SelectContent>
            </Select>

            {/* Auto-refresh toggle */}
            <div className="flex items-center gap-2">
              <Switch
                checked={autoRefresh}
                onCheckedChange={setAutoRefresh}
                className="data-[state=checked]:bg-primary"
              />
              <span className="text-xs font-mono text-muted-foreground uppercase">
                Live
              </span>
            </div>

            {/* Refresh */}
            <Button
              variant="outline"
              size="icon"
              onClick={fetchLogs}
              disabled={isLoading}
              className="border-2 border-border"
            >
              <RefreshCw
                className={`w-4 h-4 ${isLoading ? "animate-spin" : ""}`}
              />
            </Button>

            {/* Export */}
            <Button
              variant="outline"
              size="icon"
              onClick={handleExport}
              disabled={logs.length === 0}
              className="border-2 border-border"
              title="Export logs as JSON"
            >
              <Download className="w-4 h-4" />
            </Button>
          </div>
        </div>
      </div>

      {/* Level summary badges */}
      <div className="container mx-auto px-4 sm:px-6 py-3 flex flex-wrap gap-2">
        {(["ERROR", "WARN", "INFO", "DEBUG"] as const).map((lvl) => {
          const cfg = LEVEL_CONFIG[lvl];
          const count = levelCounts[lvl] ?? 0;
          return (
            <Badge
              key={lvl}
              variant="outline"
              className={`${cfg.bg} ${cfg.color} ${cfg.border} font-mono text-xs cursor-pointer hover:opacity-80 transition-opacity`}
              onClick={() =>
                setLevelFilter(
                  levelFilter === lvl.toLowerCase() ? "all" : lvl.toLowerCase()
                )
              }
            >
              <span className="flex items-center gap-1">
                {cfg.icon}
                {lvl}: {count}
              </span>
            </Badge>
          );
        })}
      </div>

      {/* Log entries */}
      <main className="flex-1 container mx-auto px-4 sm:px-6 pb-6">
        <Card className="border border-border overflow-hidden">
          {isLoading && logs.length === 0 ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
          ) : logs.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
              <FileText className="w-12 h-12 mb-4 opacity-50" />
              <p className="font-mono text-sm uppercase">No log entries found</p>
              <p className="font-mono text-xs mt-1">
                {searchTerm || levelFilter !== "all"
                  ? "Try adjusting your filters"
                  : "Logs will appear here as the server runs"}
              </p>
            </div>
          ) : (
            <ScrollArea className="max-h-[calc(100vh-320px)]">
              <div className="divide-y divide-border/30">
                {/* Date grouping */}
                {(() => {
                  let lastDate = "";
                  return logs.map((entry, i) => {
                    const date = formatDate(entry.time);
                    const showDateHeader = date !== lastDate;
                    lastDate = date;
                    return (
                      <div key={i}>
                        {showDateHeader && (
                          <div className="px-4 py-1.5 bg-muted/50 border-b border-border/50 sticky top-0 z-10">
                            <span className="text-xs font-mono text-muted-foreground uppercase">
                              {date}
                            </span>
                          </div>
                        )}
                        <LogRow entry={entry} />
                      </div>
                    );
                  });
                })()}
              </div>
            </ScrollArea>
          )}

          {/* Footer with pagination info */}
          {logs.length > 0 && (
            <div className="border-t border-border px-4 py-2 flex items-center justify-between bg-muted/20">
              <span className="text-xs font-mono text-muted-foreground">
                Showing {logs.length.toLocaleString()} of{" "}
                {total.toLocaleString()} entries
              </span>
              {hasMore && (
                <span className="text-xs font-mono text-muted-foreground">
                  (more entries available)
                </span>
              )}
            </div>
          )}
        </Card>
      </main>

      <Footer />
    </div>
  );
}

export default LogsPage;
