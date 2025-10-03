import { useState } from "react";
import {
  Settings,
  Monitor,
  Moon,
  Sun,
  Info,
  Server,
  Palette,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Theme, useTheme } from "./ThemeProvider";
import { Switch } from "@/components/ui/switch";
import { useSettings } from "@/context/settings";

export function SettingsDialog() {
  const [open, setOpen] = useState(false);
  const { theme, setTheme } = useTheme();
  const { showHiddenFiles, setShowHiddenFiles } = useSettings();

  const themeOptions = [
    { value: "light", label: "Light", icon: Sun },
    { value: "dark", label: "Dark", icon: Moon },
    { value: "system", label: "System", icon: Monitor },
  ];

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="icon" className="w-9 h-9">
          <Settings className="h-4 w-4" />
          <span className="sr-only">Settings</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px] w-[400px] bg-card border-2 border-border">
        <DialogHeader>
          <DialogTitle className="text-xl font-bold font-mono uppercase tracking-wide text-foreground flex items-center gap-2">
            <Settings className="w-5 h-5" />
            Settings
          </DialogTitle>
          <DialogDescription className="text-muted-foreground font-mono text-sm">
            CONFIGURE
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Theme Settings */}
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Palette className="w-4 h-4 text-primary" />
              <h3 className="text-lg font-semibold font-mono uppercase tracking-wide text-foreground">
                Appearance
              </h3>
            </div>

            <Card className="p-4 bg-card border border-border">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-mono text-foreground/80">
                    THEME
                  </span>
                  <Badge variant="outline" className="font-mono  text-xs">
                    {theme.toUpperCase()}
                  </Badge>
                </div>

                <div className="grid grid-cols-3 gap-2">
                  {themeOptions.map((option) => {
                    const Icon = option.icon;
                    const isActive = theme === option.value;
                    return (
                      <Button
                        key={option.value}
                        variant={isActive ? "default" : "outline"}
                        size="sm"
                        onClick={() => setTheme(option.value as Theme)}
                        className={`flex items-center gap-2 font-mono text-xs ${
                          isActive
                            ? "bg-primary text-primary-foreground border-border "
                            : "hover:bg-primary"
                        }`}
                      >
                        <Icon className="w-3 h-3" />
                        {option.label.toUpperCase()}
                      </Button>
                    );
                  })}
                </div>
              </div>
            </Card>
          </div>

          {/* Server Info */}
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Server className="w-4 h-4 text-primary" />
              <h3 className="text-lg font-semibold font-mono uppercase tracking-wide text-foreground">
                Configurations
              </h3>
            </div>

            <Card className="p-4 bg-card border border-border">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-mono text-foreground/80">
                    SHOW HIDDEN FILES
                  </span>
                  <Switch
                    checked={showHiddenFiles}
                    onCheckedChange={setShowHiddenFiles}
                  />
                </div>
                {/* <div className="flex items-center justify-between">
                  <span className="text-sm font-mono text-foreground/80">PROTOCOL</span>
                  <Badge variant="outline" className="font-mono text-xs">
                    HTTP
                  </Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm font-mono text-foreground/80">STATUS</span>
                  <Badge className="bg-accent/20 text-accent border-accent/30 font-mono text-xs">
                    ACTIVE
                  </Badge>
                </div> */}
              </div>
            </Card>
          </div>

          {/* About */}
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Info className="w-4 h-4 text-primary" />
              <h3 className="text-lg font-semibold font-mono uppercase tracking-wide text-foreground">
                About
              </h3>
            </div>

            <Card className="p-4 bg-card border border-border">
              <div className="space-y-2">
                <h4 className="font-bold font-mono text-sm uppercase tracking-wide text-foreground">
                  Credits
                </h4>
                <p className="text-xs font-mono text-muted-foreground leading-relaxed">
                  Developed by{" "}
                  <a
                    href="https://github.com/tacherasasi"
                    target="_blank"
                    className="underline"
                  >
                    Tacherasasi
                  </a>
                  .
                </p>
                <div className="pt-2">
                  <Badge variant="outline" className="font-mono text-xs">
                    FILE SERVER
                  </Badge>
                </div>
              </div>
            </Card>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
