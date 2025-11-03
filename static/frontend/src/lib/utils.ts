import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import {
  FileText,
  Image as ImageIcon,
  FileSpreadsheet,
  FileCode,
  FileVideo,
  Music,
  Database,
} from "lucide-react";
import { 
  FaCss3Alt, 
  FaFileAlt, 
  FaFileArchive, 
  FaFileAudio, 
  FaFileExcel, 
  FaFileImage, 
  FaFilePdf, 
  FaFileVideo, 
  FaFileWord, 
  FaHtml5, 
  FaJava, 
  FaJs, 
  FaPhp, 
  FaPython, 
  FaReact, 
  FaVuejs,
  FaGitAlt,
  FaMarkdown,
  FaDocker
} from "react-icons/fa";
import { FaGolang } from "react-icons/fa6";
import { 
  SiRuby, 
  SiTypescript, 
  SiJson, 
  SiRust, 
  SiSvelte,
  SiKotlin,
  SiSwift,
  SiCplusplus,
  SiC,
  SiYaml,
  SiXml,
  SiSqlite
} from "react-icons/si";
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

import React from "react";

export const getFileIcon = (filename: string, className: string = "w-4 h-4 text-primary-foreground"): React.ReactElement => {
  const ext = filename.split(".").pop()?.toLowerCase();
  
  switch (ext) {
    // Image files
    case "jpg":
    case "jpeg":
      return React.createElement(FaFileImage, { className });
    case "png":
      return React.createElement(FaFileImage, { className });
    case "gif":
    case "webp":
    case "svg":
      return React.createElement(ImageIcon, { className });

    // Document files
    case "pdf":
      return React.createElement(FaFilePdf, { className });
    case "doc":
    case "docx":
      return React.createElement(FaFileWord, { className });
    case "txt":
      return React.createElement(FaFileAlt, { className });
    case "md":
      return React.createElement(FaMarkdown, { className });

    // Spreadsheet files
    case "xls":
    case "xlsx":
      return React.createElement(FaFileExcel, { className });
    case "csv":
      return React.createElement(FileSpreadsheet, { className });

    // Archive files
    case "zip":
    case "rar":
    case "tar":
    case "gz":
    case "7z":
    case "bz2":
    case "xz":
      return React.createElement(FaFileArchive, { className });

    // Code files
    case "js":
      return React.createElement(FaJs, { className });
    case "ts":
    case "tsx":
      return React.createElement(SiTypescript, { className });
    case "jsx":
      return React.createElement(FaReact, { className });
    case "py":
      return React.createElement(FaPython, { className });
    case "java":
      return React.createElement(FaJava, { className });
    case "go":
      return React.createElement(FaGolang, { className });
    case "php":
      return React.createElement(FaPhp, { className });
    case "rb":
      return React.createElement(SiRuby, { className });
    case "html":
      return React.createElement(FaHtml5, { className });
    case "css":
    case "scss":
    case "sass":
      return React.createElement(FaCss3Alt, { className });
    case "json":
      return React.createElement(SiJson, { className });
    case "xml":
      return React.createElement(SiXml, { className });
    case "yml":
    case "yaml":
      return React.createElement(SiYaml, { className });

    // Media files
    case "mp4":
    case "mkv":
    case "avi":
    case "mov":
    case "wmv":
    case "flv":
    case "webm":
      return React.createElement(FaFileVideo, { className });
    case "mp3":
    case "wav":
    case "ogg":
    case "flac":
    case "aac":
      return React.createElement(FaFileAudio, { className });

    // Other code files
    case "c":
      return React.createElement(SiC, { className });
    case "cpp":
    case "cc":
    case "cxx":
    case "hpp":
    case "h":
      return React.createElement(SiCplusplus, { className });
    case "rs":
      return React.createElement(SiRust, { className });
    case "vue":
      return React.createElement(FaVuejs, { className });
    case "svelte":
      return React.createElement(SiSvelte, { className });
    case "kt":
    case "kts":
      return React.createElement(SiKotlin, { className });
    case "swift":
      return React.createElement(SiSwift, { className });
    case "sql":
    case "db":
    case "sqlite":
      return React.createElement(SiSqlite, { className });
    case "dockerfile":
      return React.createElement(FaDocker, { className });
    case "gitignore":
    case "gitattributes":
      return React.createElement(FaGitAlt, { className });

    default:
      return React.createElement(FileText, { className });
  }
};
