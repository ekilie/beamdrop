import { useState, useEffect } from "react";
import {
  BarChart3,
  Server,
  Upload,
  Download,
  HardDrive,
  TrendingUp,
  Clock,
  Activity,
  Database,
  Settings,
} from "lucide-react";
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

interface SidebarStats {
  downloads: number;
  uploads: number;
  startTime: string;
}

interface AppSidebarProps {
  password?: string;
}

export function AppSidebar({ password = "" }: AppSidebarProps) {
  const { state } = useSidebar();
  const [stats, setStats] = useState<SidebarStats>({
    downloads: 0,
    uploads: 0,
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

      wStatus.onmessage = (e) =>{
        const data = JSON.parse(e.data);
        console.log("WebSocket stats data:", data);
        setStats({
          downloads: data.total_downloads || 0,
          uploads: data.total_uploads || 0,
          startTime: stats.startTime,
        });
        console.log("Updated stats via WebSocket:", stats);
      }

      wStatus.onerror = (e) => {
        console.error("WebSocket error:", e);
      }
    } catch (error) {
      console.error("Failed to fetch stats:", error);
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

        {!isCollapsed && (
          <>
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
          <div className="space-y-4">
            <div className="flex items-center justify-center gap-2">
              <ThemeToggle />
              <SettingsDialog />
            </div>
            <div className="text-center">
              <p className="text-xs font-mono text-sidebar-foreground/60">
                DROP IT. BEAM IT. DONE.
              </p>
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-2">
            <ThemeToggle />
            <SettingsDialog />
          </div>
        )}
      </SidebarFooter>
    </Sidebar>
  );
}
