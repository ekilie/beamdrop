import React from 'react'

const Footer = () => {
  return (
      <footer className="mt-16 border-t-2 border-border bg-card animate-fade-in">
        <div className="container mx-auto px-4 sm:px-6 py-6">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
            <p className="font-mono text-xs text-muted-foreground uppercase tracking-wide text-center sm:text-left">
              Crafted By • <a href="http://github.com/tacheraSasi" className="text-primary hover:underline transition-smooth" target="_blank" rel="noopener noreferrer">Tachera Sasi</a>
            </p>
            <p className="font-mono text-xs text-muted-foreground uppercase tracking-wide">
              BeamDrop • File Server
            </p>
          </div>
        </div>
      </footer>
  )
}

export default Footer
