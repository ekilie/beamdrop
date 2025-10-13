import { motion } from "framer-motion";
import { Cloud, Upload, FolderOpen } from "lucide-react";

interface EmptyStateProps {
  searchTerm?: string;
  onUploadClick?: () => void;
}

export const EmptyState: React.FC<EmptyStateProps> = ({ searchTerm, onUploadClick }) => {
  if (searchTerm) {
    return (
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex flex-col items-center justify-center py-16"
      >
        <motion.div
          animate={{
            rotate: [0, -10, 10, -10, 0],
          }}
          transition={{
            duration: 2,
            repeat: Infinity,
            repeatDelay: 3,
          }}
          className="mb-6"
        >
          <div className="w-32 h-32 rounded-full bg-muted/50 flex items-center justify-center">
            <FolderOpen className="w-16 h-16 text-muted-foreground" />
          </div>
        </motion.div>
        <h3 className="text-xl font-mono font-bold text-foreground mb-2">
          NO MATCHING FILES
        </h3>
        <p className="text-muted-foreground font-mono text-sm">
          Try a different search term
        </p>
      </motion.div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.5 }}
      className="flex flex-col items-center justify-center py-16"
    >
      <motion.div
        animate={{
          y: [0, -10, 0],
        }}
        transition={{
          duration: 3,
          repeat: Infinity,
          ease: "easeInOut",
        }}
        className="mb-8 relative"
      >
        <motion.div
          animate={{
            scale: [1, 1.1, 1],
            opacity: [0.5, 0.8, 0.5],
          }}
          transition={{
            duration: 2,
            repeat: Infinity,
            ease: "easeInOut",
          }}
          className="absolute inset-0 bg-primary/20 rounded-full blur-xl"
        />
        <div className="w-32 h-32 rounded-full bg-gradient-to-br from-primary/20 to-accent/20 flex items-center justify-center relative">
          <Cloud className="w-16 h-16 text-primary" />
        </div>
      </motion.div>

      <motion.h3
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.2 }}
        className="text-2xl font-mono font-bold text-foreground mb-2"
      >
        NO FILES YET
      </motion.h3>
      
      <motion.p
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.3 }}
        className="text-muted-foreground font-mono text-sm mb-8 text-center max-w-md"
      >
        Upload your first file to get started with your cloud storage
      </motion.p>

      {onUploadClick && (
        <motion.button
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={onUploadClick}
          className="flex items-center gap-2 px-6 py-3 bg-primary text-primary-foreground rounded-lg font-mono font-bold uppercase tracking-wide hover:bg-primary/90 transition-colors"
        >
          <Upload className="w-5 h-5" />
          Upload Files
        </motion.button>
      )}
    </motion.div>
  );
};
