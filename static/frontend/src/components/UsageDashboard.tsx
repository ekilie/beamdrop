import { useState, useEffect } from "react";
import {
  HardDrive,
  Download,
  Upload,
  Activity,
  TrendingUp,
  BarChart3,
  Clock,
  RefreshCw,
  ArrowUpFromLine,
  ArrowDownToLine,
} from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";

interface SystemStats {
  memory: {
    total: number;
    available: number;
    used: number;
    percent: number;
  };
  disk: {
    total: number;
    free: number;
    used: number;
    percent: number;
  };
  cpu: {
    cores: number;
    goroutines: number;
  };
}

interface UsageStats {
  downloads: number;
  uploads: number;
  requests: number;
  bytesUploaded: number;
  bytesDownloaded: number;
  startTime: string;
  system?: SystemStats;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return `${size.toFixed(1)} ${units[unitIndex]}`;
}

function calculateUptime(startTime: string): string {
  const start = new Date(startTime);
  const now = new Date();
  const diff = now.getTime() - start.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days > 0) return `${days}d ${hours % 24}h ${minutes % 60}m`;
  if (hours > 0) return `${hours}h ${minutes % 60}m`;
  return `${minutes}m`;
}

interface StatCardProps {
  title: string;
  value: string;
  subtitle?: string;
  icon: React.ReactNode;
  accent?: string;
  badge?: string;
}

function StatCard({ title, value, subtitle, icon, accent = "text-primary", badge }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-mono uppercase text-muted-foreground">
            {title}
          </CardTitle>
          <div className={`p-2 rounded-lg bg-muted ${accent}`}>{icon}</div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold font-mono">{value}</div>
        {subtitle && (
          <p className="text-xs text-muted-foreground mt-1">{subtitle}</p>
        )}
        {badge && (
          <Badge variant="outline" className="mt-2 text-xs">
            {badge}
          </Badge>
        )}
      </CardContent>
    </Card>
  );
}

interface ProgressCardProps {
  title: string;
  used: number;
  total: number;
  percent: number;
  icon: React.ReactNode;
  color: string;
  bgColor: string;
}

function ProgressCard({ title, used, total, percent, icon, color, bgColor }: ProgressCardProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-mono uppercase text-muted-foreground">
            {title}
          </CardTitle>
          <div className={`p-2 rounded-lg ${bgColor}`}>
            <div className={color}>{icon}</div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-end justify-between">
          <span className="text-2xl font-bold font-mono">
            {formatBytes(used)}
          </span>
          <span className={`text-sm font-mono font-medium ${color}`}>
            {percent.toFixed(1)}%
          </span>
        </div>
        <Progress value={Math.min(percent, 100)} className="h-2" />
        <p className="text-xs text-muted-foreground">
          {formatBytes(used)} used of {formatBytes(total)}
        </p>
      </CardContent>
    </Card>
  );
}

export function UsageDashboard() {
  const [stats, setStats] = useState<UsageStats | null>(null);
  const [uptime, setUptime] = useState("0m");
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const connectWebSocket = () => {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/stats`);

    ws.onmessage = (e) => {
      try {
        const data: UsageStats = JSON.parse(e.data);
        setStats(data);
        setLastUpdated(new Date());
      } catch {
        // ignore parse errors
      }
    };

    ws.onerror = () => {};

    return ws;
  };

  useEffect(() => {
    const ws = connectWebSocket();
    return () => ws.close();
  }, []);

  useEffect(() => {
    if (!stats?.startTime) return;
    setUptime(calculateUptime(stats.startTime));
    const interval = setInterval(() => {
      setUptime(calculateUptime(stats.startTime));
    }, 60000);
    return () => clearInterval(interval);
  }, [stats?.startTime]);

  const handleRefresh = () => {
    const ws = connectWebSocket();
    setTimeout(() => ws.close(), 2000);
  };

  return (
    <div className="p-6 space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold font-mono uppercase tracking-wide">
            Usage Dashboard
          </h1>
          <p className="text-muted-foreground mt-1">
            Real-time storage, bandwidth, and request metrics
          </p>
        </div>
        <div className="flex items-center gap-3">
          {lastUpdated && (
            <span className="text-xs font-mono text-muted-foreground">
              Updated {lastUpdated.toLocaleTimeString()}
            </span>
          )}
          <Button onClick={handleRefresh} variant="outline" size="sm" className="gap-2">
            <RefreshCw className="w-4 h-4" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Activity summary cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Total Requests"
          value={stats ? stats.requests.toLocaleString() : "—"}
          subtitle="Since server start"
          icon={<Activity className="w-4 h-4" />}
          accent="text-primary"
          badge="All-time"
        />
        <StatCard
          title="Downloads"
          value={stats ? stats.downloads.toLocaleString() : "—"}
          subtitle="Files served"
          icon={<Download className="w-4 h-4" />}
          accent="text-blue-500"
        />
        <StatCard
          title="Uploads"
          value={stats ? stats.uploads.toLocaleString() : "—"}
          subtitle="Files received"
          icon={<Upload className="w-4 h-4" />}
          accent="text-green-500"
        />
        <StatCard
          title="Server Uptime"
          value={uptime}
          subtitle={stats ? `Since ${new Date(stats.startTime).toLocaleDateString()}` : "—"}
          icon={<Clock className="w-4 h-4" />}
          accent="text-purple-500"
        />
      </div>

      {/* Bandwidth section */}
      <div>
        <h2 className="text-lg font-bold font-mono uppercase tracking-wide mb-4 flex items-center gap-2">
          <BarChart3 className="w-5 h-5 text-primary" />
          Bandwidth
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <StatCard
            title="Data Uploaded"
            value={stats ? formatBytes(stats.bytesUploaded) : "—"}
            subtitle="Total bytes received from clients"
            icon={<ArrowUpFromLine className="w-4 h-4" />}
            accent="text-green-500"
          />
          <StatCard
            title="Data Downloaded"
            value={stats ? formatBytes(stats.bytesDownloaded) : "—"}
            subtitle="Total bytes served to clients"
            icon={<ArrowDownToLine className="w-4 h-4" />}
            accent="text-blue-500"
          />
          <StatCard
            title="Total Transfer"
            value={stats ? formatBytes((stats.bytesUploaded || 0) + (stats.bytesDownloaded || 0)) : "—"}
            subtitle="Combined in + out bandwidth"
            icon={<TrendingUp className="w-4 h-4" />}
            accent="text-orange-500"
          />
        </div>
      </div>

      {/* Storage & System Resources */}
      {stats?.system && (
        <div>
          <h2 className="text-lg font-bold font-mono uppercase tracking-wide mb-4 flex items-center gap-2">
            <HardDrive className="w-5 h-5 text-primary" />
            Storage & System Resources
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <ProgressCard
              title="Disk Storage"
              used={stats.system.disk.used}
              total={stats.system.disk.total}
              percent={stats.system.disk.percent}
              icon={<HardDrive className="w-4 h-4" />}
              color="text-green-500"
              bgColor="bg-green-500/10"
            />
            <ProgressCard
              title="Memory Usage"
              used={stats.system.memory.used}
              total={stats.system.memory.total}
              percent={stats.system.memory.percent}
              icon={<Activity className="w-4 h-4" />}
              color="text-blue-500"
              bgColor="bg-blue-500/10"
            />
            <Card>
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm font-mono uppercase text-muted-foreground">
                    CPU / Runtime
                  </CardTitle>
                  <div className="p-2 rounded-lg bg-purple-500/10">
                    <Activity className="w-4 h-4 text-purple-500" />
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="text-2xl font-bold font-mono">
                  {stats.system.cpu.cores} cores
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground font-mono">Goroutines</span>
                  <Badge variant="outline" className="font-mono">
                    {stats.system.cpu.goroutines}
                  </Badge>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground font-mono">Disk Free</span>
                  <span className="font-mono text-green-500">
                    {formatBytes(stats.system.disk.free)}
                  </span>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      )}

      {/* Quick stats summary */}
      {stats && (
        <Card>
          <CardHeader>
            <CardTitle className="font-mono uppercase text-sm">Summary</CardTitle>
            <CardDescription>Overview of all activity since server start</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
              <div>
                <div className="text-2xl font-bold font-mono text-primary">{stats.requests.toLocaleString()}</div>
                <div className="text-xs text-muted-foreground font-mono uppercase mt-1">Requests</div>
              </div>
              <div>
                <div className="text-2xl font-bold font-mono text-blue-500">{stats.downloads.toLocaleString()}</div>
                <div className="text-xs text-muted-foreground font-mono uppercase mt-1">Downloads</div>
              </div>
              <div>
                <div className="text-2xl font-bold font-mono text-green-500">{stats.uploads.toLocaleString()}</div>
                <div className="text-xs text-muted-foreground font-mono uppercase mt-1">Uploads</div>
              </div>
              <div>
                <div className="text-2xl font-bold font-mono text-orange-500">
                  {formatBytes((stats.bytesUploaded || 0) + (stats.bytesDownloaded || 0))}
                </div>
                <div className="text-xs text-muted-foreground font-mono uppercase mt-1">Bandwidth</div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
