import React, { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import { Badge } from "@/components/ui/badge";
import {
  Play,
  Pause,
  Volume2,
  VolumeX,
  SkipBack,
  SkipForward,
  Music,
  Maximize,
  Minimize,
} from "lucide-react";
import { motion } from "framer-motion";

interface EnhancedAudioPlayerProps {
  src: string;
  fileName: string;
  onLoadedData: () => void;
  onError: () => void;
}

export function EnhancedAudioPlayer({
  src,
  fileName,
  onLoadedData,
  onError,
}: EnhancedAudioPlayerProps) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(100);
  const [isMuted, setIsMuted] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    const handleTimeUpdate = () => setCurrentTime(audio.currentTime);
    const handleLoadedMetadata = () => {
      setDuration(audio.duration);
      onLoadedData();
    };
    const handleEnded = () => setIsPlaying(false);

    audio.addEventListener("timeupdate", handleTimeUpdate);
    audio.addEventListener("loadedmetadata", handleLoadedMetadata);
    audio.addEventListener("ended", handleEnded);

    return () => {
      audio.removeEventListener("timeupdate", handleTimeUpdate);
      audio.removeEventListener("loadedmetadata", handleLoadedMetadata);
      audio.removeEventListener("ended", handleEnded);
    };
  }, [onLoadedData]);

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
    };
  }, []);

  const toggleFullscreen = () => {
    const container = containerRef.current;
    if (!container) return;

    if (!document.fullscreenElement) {
      container.requestFullscreen().catch((err) => {
        console.error('Error attempting to enable fullscreen:', err);
      });
    } else {
      document.exitFullscreen().catch((err) => {
        console.error('Error attempting to exit fullscreen:', err);
      });
    }
  };

  const togglePlay = () => {
    const audio = audioRef.current;
    if (!audio) return;

    if (isPlaying) {
      audio.pause();
    } else {
      audio.play();
    }
    setIsPlaying(!isPlaying);
  };

  const handleProgressChange = (value: number[]) => {
    const audio = audioRef.current;
    if (!audio) return;
    audio.currentTime = value[0];
    setCurrentTime(value[0]);
  };

  const handleVolumeChange = (value: number[]) => {
    const audio = audioRef.current;
    if (!audio) return;
    const newVolume = value[0];
    audio.volume = newVolume / 100;
    setVolume(newVolume);
    setIsMuted(newVolume === 0);
  };

  const toggleMute = () => {
    const audio = audioRef.current;
    if (!audio) return;

    if (isMuted) {
      audio.volume = volume / 100;
      setIsMuted(false);
    } else {
      audio.volume = 0;
      setIsMuted(true);
    }
  };

  const skip = (seconds: number) => {
    const audio = audioRef.current;
    if (!audio) return;
    audio.currentTime = Math.max(0, Math.min(duration, currentTime + seconds));
  };

  const formatTime = (time: number) => {
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    return `${minutes}:${seconds.toString().padStart(2, "0")}`;
  };

  const progress = duration > 0 ? (currentTime / duration) * 100 : 0;

  return (
    <div 
      ref={containerRef}
      className="w-full bg-gradient-to-br from-primary/20 via-accent/10 to-secondary rounded-lg p-6 border-2 border-border animate-fade-in"
    >
      <audio ref={audioRef} src={src} onError={onError} />

      {/* Album Art / Visualization */}
      <div className="relative mb-6">
        <motion.div
          animate={{
            scale: isPlaying ? [1, 1.05, 1] : 1,
          }}
          transition={{
            duration: 2,
            repeat: isPlaying ? Infinity : 0,
            ease: "easeInOut",
          }}
          className="w-32 h-32 mx-auto rounded-full bg-gradient-to-br from-primary to-accent flex items-center justify-center shadow-lg relative overflow-hidden"
        >
          <div className="absolute inset-0 bg-gradient-to-br from-primary/50 to-accent/50 animate-shimmer" />
          <Music className="w-16 h-16 text-primary-foreground relative z-10" />
        </motion.div>

        {/* Equalizer Bars */}
        {isPlaying && (
          <div className="flex items-end justify-center gap-1 mt-4 h-12">
            {[...Array(12)].map((_, i) => (
              <motion.div
                key={i}
                animate={{
                  height: ["20%", "100%", "40%", "80%", "20%"],
                }}
                transition={{
                  duration: 0.8 + Math.random() * 0.4,
                  repeat: Infinity,
                  ease: "easeInOut",
                  delay: i * 0.05,
                }}
                className="w-1.5 rounded-full bg-gradient-to-t from-primary to-accent"
              />
            ))}
          </div>
        )}
      </div>

      {/* Track Info */}
      <div className="text-center mb-6">
        <h3 className="font-mono font-bold text-foreground text-lg truncate mb-2">
          {fileName}
        </h3>
        <Badge variant="secondary" className="font-mono text-xs">
          AUDIO TRACK
        </Badge>
      </div>

      {/* Progress Bar */}
      <div className="mb-4">
        <Slider
          value={[currentTime]}
          max={duration || 100}
          step={0.1}
          onValueChange={handleProgressChange}
          className="cursor-pointer"
        />
        <div className="flex justify-between text-xs text-muted-foreground mt-2 font-mono">
          <span>{formatTime(currentTime)}</span>
          <span>{formatTime(duration)}</span>
        </div>
      </div>

      {/* Controls */}
      <div className="flex items-center justify-center gap-2 mb-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => skip(-10)}
          className="h-10 w-10 text-foreground hover:text-primary"
        >
          <SkipBack className="h-5 w-5" />
        </Button>

        <Button
          variant="default"
          size="icon"
          onClick={togglePlay}
          className="h-14 w-14 rounded-full shadow-lg hover:scale-105 transition-transform"
        >
          {isPlaying ? (
            <Pause className="h-6 w-6" />
          ) : (
            <Play className="h-6 w-6 ml-0.5" />
          )}
        </Button>

        <Button
          variant="ghost"
          size="icon"
          onClick={() => skip(10)}
          className="h-10 w-10 text-foreground hover:text-primary"
        >
          <SkipForward className="h-5 w-5" />
        </Button>
      </div>

      {/* Volume Control and Fullscreen */}
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleMute}
          className="h-8 w-8 text-foreground hover:text-primary flex-shrink-0"
        >
          {isMuted || volume === 0 ? (
            <VolumeX className="h-4 w-4" />
          ) : (
            <Volume2 className="h-4 w-4" />
          )}
        </Button>
        <Slider
          value={[isMuted ? 0 : volume]}
          max={100}
          step={1}
          onValueChange={handleVolumeChange}
          className="flex-1"
        />
        <span className="text-xs font-mono text-muted-foreground w-10 text-right">
          {Math.round(isMuted ? 0 : volume)}%
        </span>
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleFullscreen}
          className="h-8 w-8 text-foreground hover:text-primary flex-shrink-0"
        >
          {isFullscreen ? (
            <Minimize className="h-4 w-4" />
          ) : (
            <Maximize className="h-4 w-4" />
          )}
        </Button>
      </div>
    </div>
  );
}
