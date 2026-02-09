import { useState, useEffect } from "react";
import {
  Server,
  Upload,
  Download,
  HardDrive,
  TrendingUp,
  Clock,
  Activity,
  Cpu,
  MemoryStick,
  Key,
  Home,
  Share2,
} from "lucide-react";
import { useNavigate, useLocation } from "react-router-dom";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarHeader,
  SidebarFooter,
  useSidebar,
} from "@/components/ui/sidebar";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { ThemeToggle } from "@/components/ThemeToggle";
import { SettingsDialog } from "@/components/SettingsDialog";

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

interface SidebarStats {
  downloads: number;
  uploads: number;
  requests: number;
  startTime: string;
  system?: SystemStats;
}

interface AppSidebarProps {
  password?: string;
}

export function AppSidebar({ password = "" }: AppSidebarProps) {
  const { state } = useSidebar();
  const navigate = useNavigate();
  const location = useLocation();
  const [stats, setStats] = useState<SidebarStats>({
    downloads: 0,
    uploads: 0,
    requests: 0,
    startTime: new Date().toISOString(),
  });
  const [uptime, setUptime] = useState("");

  const fetchStats = async () => {
    try {
      const headers: Record<string, string> = {};
      if (password) {
        headers["X-Password"] = password;
      }

      const wStatus = new WebSocket(`ws://${window.location.host}/ws/stats`);

      wStatus.onmessage = (e) => {
        const data = JSON.parse(e.data);
        const updatedStats: SidebarStats = {
          downloads: data.downloads || 0,
          uploads: data.uploads || 0,
          requests: data.requests || 0,
          startTime: data.startTime || new Date().toISOString(),
          system: data.system,
        };
        setStats(updatedStats);
      };

      wStatus.onerror = (e) => {
      };
    } catch (error) {
      console.error("Err", error);
    }
  };

  const calculateUptime = () => {
    if (!stats.startTime) return "0m";

    const start = new Date(stats.startTime);
    const now = new Date();
    const diff = now.getTime() - start.getTime();

    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ${hours % 24}h`;
    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    return `${minutes}m`;
  };

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 30000); // Updates every 30 seconds
    return () => clearInterval(interval);
  }, [password]);

  useEffect(() => {
    const uptimeInterval = setInterval(() => {
      setUptime(calculateUptime());
    }, 60000); // Updates uptime every minute

    setUptime(calculateUptime());
    return () => clearInterval(uptimeInterval);
  }, [stats.startTime]);

  const isCollapsed = state === "collapsed";

  // Helper function to format bytes
  const formatBytes = (bytes: number): string => {
    const units = ["B", "KB", "MB", "GB", "TB"];
    let size = bytes;
    let unitIndex = 0;

    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }

    return `${size.toFixed(1)} ${units[unitIndex]}`;
  };

  return (
    <Sidebar className="border-r border-sidebar-border bg-sidebar">
      <SidebarHeader className="p-4">
        <div className="flex items-center gap-3">
          <div className="bg-primary p-2 rounded border border-primary">
            <Server className="w-5 h-5 text-primary-foreground" />
          </div>
          {!isCollapsed && (
            <div>
              <h2 className="text-lg font-bold font-mono uppercase tracking-wide text-sidebar-foreground">
                beamdrop
              </h2>
              <p className="text-sidebar-foreground/60 font-mono text-xs">
                FILE SERVER
              </p>
            </div>
          )}
        </div>
      </SidebarHeader>

      <SidebarContent className="px-4">
        {/* Server Status */}
        <SidebarGroup>
          <SidebarGroupLabel className="text-sidebar-foreground/80 font-mono text-xs">
            {isCollapsed ? "STATUS" : "SERVER STATUS"}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <div className="space-y-3">
              <Card className="p-3 bg-sidebar-accent border border-sidebar-border">
                <div className="flex items-center gap-2">
                  <Activity className="w-4 h-4 text-accent" />
                  {!isCollapsed && (
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-mono text-sidebar-foreground">
                          ONLINE
                        </span>
                        <Badge
                          variant="secondary"
                          className="bg-accent/20 text-accent border-accent/30"
                        >
                          ACTIVE
                        </Badge>
                      </div>
                    </div>
                  )}
                </div>
              </Card>

              {!isCollapsed && (
                <div className="grid grid-cols-1 gap-2">
                  <div className="flex items-center justify-between p-2 rounded bg-sidebar-accent/50">
                    <div className="flex items-center gap-2">
                      <Clock className="w-4 h-4 text-sidebar-foreground/60" />
                      <span className="text-xs font-mono text-sidebar-foreground/80">
                        UPTIME
                      </span>
                    </div>
                    <span className="text-xs font-mono font-medium text-sidebar-foreground">
                      {uptime}
                    </span>
                  </div>
                </div>
              )}
            </div>
          </SidebarGroupContent>
        </SidebarGroup>

        <Separator className="my-4 bg-sidebar-border" />

        {/* Analytics */}
        <SidebarGroup>
          <SidebarGroupLabel className="text-sidebar-foreground/80 font-mono text-xs">
            {isCollapsed ? "STATS" : "ANALYTICS"}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton className="p-3 bg-sidebar-accent/30 hover:bg-sidebar-accent">
                  <div className="flex items-center gap-3 w-full">
                    <div className="bg-primary/10 p-2 rounded">
                      <Download className="w-4 h-4 text-primary" />
                    </div>
                    {!isCollapsed && (
                      <div className="flex-1">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-mono text-sidebar-foreground/80">
                            DOWNLOADS
                          </span>
                          <Badge
                            variant="outline"
                            className="bg-primary/10 text-primary border-primary/30"
                          >
                            {stats.downloads}
                          </Badge>
                        </div>
                      </div>
                    )}
                  </div>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton className="p-3 bg-sidebar-accent/30 hover:bg-sidebar-accent">
                  <div className="flex items-center gap-3 w-full">
                    <div className="bg-accent/10 p-2 rounded">
                      <Upload className="w-4 h-4 text-accent" />
                    </div>
                    {!isCollapsed && (
                      <div className="flex-1">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-mono text-sidebar-foreground/80">
                            UPLOADS
                          </span>
                          <Badge
                            variant="outline"
                            className="bg-accent/10 text-accent border-accent/30"
                          >
                            {stats.uploads}
                          </Badge>
                        </div>
                      </div>
                    )}
                  </div>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton className="p-3 bg-sidebar-accent/30 hover:bg-sidebar-accent">
                  <div className="flex items-center gap-3 w-full">
                    <div className="bg-muted-foreground/10 p-2 rounded">
                      <TrendingUp className="w-4 h-4 text-muted-foreground" />
                    </div>
                    {!isCollapsed && (
                      <div className="flex-1">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-mono text-sidebar-foreground/80">
                            TOTAL OPS
                          </span>
                          <Badge
                            variant="outline"
                            className="bg-muted/20 text-muted-foreground border-muted"
                          >
                            {stats.downloads + stats.uploads}
                          </Badge>
                        </div>
                      </div>
                    )}
                  </div>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {/* System Resources */}
        {stats.system && (
          <>
            <Separator className="my-4 bg-sidebar-border" />
            <SidebarGroup>
              <SidebarGroupLabel className="text-sidebar-foreground/80 font-mono text-xs">
                {isCollapsed ? "SYSTEM" : "SYSTEM RESOURCES"}
              </SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {/* Memory */}
                  <SidebarMenuItem>
                    <SidebarMenuButton className="p-3 bg-sidebar-accent/30 hover:bg-sidebar-accent h-auto">
                      <div className="flex items-start gap-3 w-full min-w-0">
                        <div className="bg-blue-500/10 p-2 rounded flex-shrink-0">
                          <MemoryStick className="w-4 h-4 text-blue-500" />
                        </div>
                        {!isCollapsed && (
                          <div className="flex-1 min-w-0 space-y-1.5">
                            <div className="flex items-center justify-between gap-2">
                              <span className="text-sm font-mono text-sidebar-foreground/80 whitespace-nowrap">
                                MEMORY
                              </span>
                              <Badge
                                variant="outline"
                                className="bg-blue-500/10 text-blue-500 border-blue-500/30 flex-shrink-0"
                              >
                                {stats.system.memory.percent.toFixed(1)}%
                              </Badge>
                            </div>
                            <div className="w-full bg-sidebar-border rounded-full h-1.5">
                              <div
                                className="bg-blue-500 h-1.5 rounded-full transition-all"
                                style={{
                                  width: `${Math.min(stats.system.memory.percent, 100)}%`,
                                }}
                              />
                            </div>
                            <span className="text-xs font-mono text-sidebar-foreground/60 block truncate">
                              {formatBytes(stats.system.memory.used)} / {formatBytes(stats.system.memory.total)}
                            </span>
                          </div>
                        )}
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>

                  {/* Disk */}
                  <SidebarMenuItem>
                    <SidebarMenuButton className="p-3 bg-sidebar-accent/30 hover:bg-sidebar-accent h-auto">
                      <div className="flex items-start gap-3 w-full min-w-0">
                        <div className="bg-green-500/10 p-2 rounded flex-shrink-0">
                          <HardDrive className="w-4 h-4 text-green-500" />
                        </div>
                        {!isCollapsed && (
                          <div className="flex-1 min-w-0 space-y-1.5">
                            <div className="flex items-center justify-between gap-2">
                              <span className="text-sm font-mono text-sidebar-foreground/80 whitespace-nowrap">
                                DISK
                              </span>
                              <Badge
                                variant="outline"
                                className="bg-green-500/10 text-green-500 border-green-500/30 flex-shrink-0"
                              >
                                {stats.system.disk.percent.toFixed(1)}%
                              </Badge>
                            </div>
                            <div className="w-full bg-sidebar-border rounded-full h-1.5">
                              <div
                                className="bg-green-500 h-1.5 rounded-full transition-all"
                                style={{
                                  width: `${Math.min(stats.system.disk.percent, 100)}%`,
                                }}
                              />
                            </div>
                            <span className="text-xs font-mono text-sidebar-foreground/60 block truncate">
                              {formatBytes(stats.system.disk.used)} / {formatBytes(stats.system.disk.total)}
                            </span>
                          </div>
                        )}
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>

                  {/* CPU */}
                  <SidebarMenuItem>
                    <SidebarMenuButton className="p-3 bg-sidebar-accent/30 hover:bg-sidebar-accent h-auto">
                      <div className="flex items-start gap-3 w-full min-w-0">
                        <div className="bg-purple-500/10 p-2 rounded flex-shrink-0">
                          <Cpu className="w-4 h-4 text-purple-500" />
                        </div>
                        {!isCollapsed && (
                          <div className="flex-1 min-w-0 space-y-1.5">
                            <div className="flex items-center justify-between gap-2">
                              <span className="text-sm font-mono text-sidebar-foreground/80 whitespace-nowrap">
                                CPU CORES
                              </span>
                              <Badge
                                variant="outline"
                                className="bg-purple-500/10 text-purple-500 border-purple-500/30 flex-shrink-0"
                              >
                                {stats.system.cpu.cores}
                              </Badge>
                            </div>
                            <span className="text-xs font-mono text-sidebar-foreground/60 block truncate">
                              {stats.system.cpu.goroutines} goroutines
                            </span>
                          </div>
                        )}
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </>
        )}

        {!isCollapsed && (
          <>
            <Separator className="my-4 bg-sidebar-border" />

            {/* Navigation */}
            <SidebarGroup>
              <SidebarGroupLabel className="text-sidebar-foreground/80 font-mono text-xs">
                NAVIGATION
              </SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      onClick={() => navigate("/")}
                      className={`hover:bg-sidebar-accent ${location.pathname === "/" ? "bg-sidebar-accent" : ""}`}
                    >
                      <Home className="w-4 h-4" />
                      <span className="font-mono text-sm">HOME</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      onClick={() => navigate("/shares")}
                      className={`hover:bg-sidebar-accent ${location.pathname === "/shares" ? "bg-sidebar-accent" : ""}`}
                    >
                      <Share2 className="w-4 h-4" />
                      <span className="font-mono text-sm">SHARES</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      onClick={() => navigate("/api-keys")}
                      className={`hover:bg-sidebar-accent ${location.pathname === "/api-keys" ? "bg-sidebar-accent" : ""}`}
                    >
                      <Key className="w-4 h-4" />
                      <span className="font-mono text-sm">API KEYS</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>

            <Separator className="my-4 bg-sidebar-border" />

            {/* Quick Actions */}
            <SidebarGroup>
              <SidebarGroupLabel className="text-sidebar-foreground/80 font-mono text-xs">
                QUICK ACTIONS
              </SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      onClick={() => {
                        const uploadSection = document.querySelector(
                          "[data-upload-section]"
                        );
                        uploadSection?.scrollIntoView({ behavior: "smooth" });
                      }}
                      className="hover:bg-sidebar-accent"
                    >
                      <Upload className="w-4 h-4" />
                      <span className="font-mono text-sm">UPLOAD FILES</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      onClick={() => {
                        const filesSection = document.querySelector(
                          "[data-files-section]"
                        );
                        filesSection?.scrollIntoView({ behavior: "smooth" });
                      }}
                      className="hover:bg-sidebar-accent"
                    >
                      <HardDrive className="w-4 h-4" />
                      <span className="font-mono text-sm">BROWSE FILES</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </>
        )}
      </SidebarContent>

      <SidebarFooter className="p-4">
        {!isCollapsed ? (
          <div className="space-y-4 w-full">
            <div className="flex items-center justify-center gap-2 w-full">
              <ThemeToggle />
              <SettingsDialog />
            </div>
            <div className="text-center w-full">
              <p className="text-xs font-mono text-sidebar-foreground/60">
                DROP IT. BEAM IT. DONE.
              </p>
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center gap-2 w-full">
            <ThemeToggle />
            <SettingsDialog />
          </div>
        )}
      </SidebarFooter>
    </Sidebar>
  );
}
