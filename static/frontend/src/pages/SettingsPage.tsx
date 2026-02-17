import {
    Settings,
    Monitor,
    Moon,
    Sun,
    Info,
    Server,
    Palette,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Theme, useTheme } from "@/components/ThemeProvider";
import { Switch } from "@/components/ui/switch";
import { useSettings } from "@/context/settings";

export default function SettingsPage() {
    const { theme, setTheme } = useTheme();
    const { showHiddenFiles, setShowHiddenFiles } = useSettings();

    const themeOptions = [
        { value: "light", label: "Light", icon: Sun },
        { value: "dark", label: "Dark", icon: Moon },
        { value: "system", label: "System", icon: Monitor },
    ];

    return (
        <div className="max-w-2xl mx-auto p-6 space-y-8 animate-fade-in">
            {/* Page Header */}
            <div>
                <h1 className="text-2xl font-bold font-mono uppercase tracking-wide text-foreground flex items-center gap-3">
                    <Settings className="w-6 h-6" />
                    Settings
                </h1>
                <p className="text-muted-foreground font-mono text-sm mt-1">
                    CONFIGURE YOUR BEAMDROP INSTANCE
                </p>
            </div>

            <Separator />

            {/* Appearance */}
            <section className="space-y-4">
                <div className="flex items-center gap-2">
                    <Palette className="w-5 h-5 text-primary" />
                    <h2 className="text-lg font-semibold font-mono uppercase tracking-wide text-foreground">
                        Appearance
                    </h2>
                </div>

                <Card className="p-5 bg-card border border-border">
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <span className="text-sm font-mono text-foreground/80">
                                THEME
                            </span>
                            <Badge variant="outline" className="font-mono text-xs">
                                {theme.toUpperCase()}
                            </Badge>
                        </div>

                        <div className="grid grid-cols-3 gap-3">
                            {themeOptions.map((option) => {
                                const Icon = option.icon;
                                const isActive = theme === option.value;
                                return (
                                    <Button
                                        key={option.value}
                                        variant={isActive ? "default" : "outline"}
                                        size="sm"
                                        onClick={() => setTheme(option.value as Theme)}
                                        className={`flex items-center gap-2 font-mono text-xs h-10 ${isActive
                                                ? "bg-primary text-primary-foreground border-border"
                                                : "hover:bg-primary"
                                            }`}
                                    >
                                        <Icon className="w-4 h-4" />
                                        {option.label.toUpperCase()}
                                    </Button>
                                );
                            })}
                        </div>
                    </div>
                </Card>
            </section>

            {/* Configurations */}
            <section className="space-y-4">
                <div className="flex items-center gap-2">
                    <Server className="w-5 h-5 text-primary" />
                    <h2 className="text-lg font-semibold font-mono uppercase tracking-wide text-foreground">
                        Configurations
                    </h2>
                </div>

                <Card className="p-5 bg-card border border-border">
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <div>
                                <span className="text-sm font-mono text-foreground/80 block">
                                    SHOW HIDDEN FILES
                                </span>
                                <span className="text-xs font-mono text-muted-foreground">
                                    Display files and folders starting with a dot
                                </span>
                            </div>
                            <Switch
                                checked={showHiddenFiles}
                                onCheckedChange={setShowHiddenFiles}
                            />
                        </div>
                    </div>
                </Card>
            </section>

            {/* About */}
            <section className="space-y-4">
                <div className="flex items-center gap-2">
                    <Info className="w-5 h-5 text-primary" />
                    <h2 className="text-lg font-semibold font-mono uppercase tracking-wide text-foreground">
                        About
                    </h2>
                </div>

                <Card className="p-5 bg-card border border-border">
                    <div className="space-y-3">
                        <h4 className="font-bold font-mono text-sm uppercase tracking-wide text-foreground">
                            Credits
                        </h4>
                        <p className="text-sm font-mono text-muted-foreground leading-relaxed">
                            Developed by{" "}
                            <a
                                href="https://github.com/tacherasasi"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="underline text-primary hover:text-primary/80 transition-colors"
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
            </section>
        </div>
    );
}
