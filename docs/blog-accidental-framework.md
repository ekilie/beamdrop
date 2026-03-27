# I Think I Accidentally Built a Full-Stack Framework?

_Look, I just wanted to share some files._

## The Innocent Beginning

It started, as these things always do, with a simple problem: I needed to share files between my devices. "I'll just write a quick Go server," I said. "It'll take an afternoon," I said. "It'll be like 200 lines of code," I said.

**Reader, it was not 200 lines of code.**

## What Is BeamDrop, Actually?

[BeamDrop](https://github.com/ekilie/beamdrop) is a self-hosted file sharing server. You run one command, and boom you've got a beautiful web interface for uploading, downloading, moving, copying, renaming, and searching files. It's like Dropbox, if Dropbox was a single binary and didn't sell your data to train AI models. _(No offense, Dropbox. Actually, some offense.)_

Here's the pitch: `beamdrop -dir /path/to/stuff -p mysecretpassword` and you're live. QR code on the terminal so your phone can connect. Password protection. Shareable links with expiry. WebSocket-powered real-time stats. An S3-compatible API so your CI/CD pipeline can talk to it.

One binary. No Docker required. No Node.js runtime. No "please install these 47 dependencies." Just a single executable that runs on macOS, Linux, and Windows.

## How We Got Here

### Stage 1: The Go Server (Day 1)

"I'll embed the frontend right into the Go binary using `embed.FS`. Single binary deployment. I'm a genius."

### Stage 2: The React Frontend (Week 1)

"It needs a UI. I'll use React. And Vite. And TypeScript. And Tailwind. And shadcn/ui. For a file server. This is totally proportional."

### Stage 3: The Database (Week 2)

"I need shareable links, so I need a database. SQLite is fine. I'll use GORM. Oh wait, the SQLite driver needs CGO and that breaks cross-compilation. Let me swap to a pure-Go SQLite implementation so the binary stays fully static. This is still a simple file server."

### Stage 4: The S3 API (Week 3)

"What if it also spoke S3? Like, the actual S3 API. With buckets and objects and presigned URLs. For a file server that I built because I wanted to share a PDF with my phone."

### Stage 5: The Realization (Now)

I stared at my codebase. A Go backend with embedded frontend assets. React with client-side routing. SPA fallback handling. SQLite with migrations. WebSocket connections. JWT authentication. API key management. CORS middleware. Security headers. TLS support. Cross-platform builds.

_I built a framework._

I didn't mean to. I was just solving problems one at a time. "Oh, the frontend routes break on refresh because Go doesn't know about React Router" so I added SPA fallback. "Oh, `CGO_ENABLED=0` breaks SQLite" so I swapped to a pure-Go driver. "Oh, I need auth" JWT and password hashing. Each fix was reasonable in isolation. Together, they formed a full-stack application framework that compiles to a zero-dependency binary.

## The Architecture (That I Definitely Planned From The Start)

```
┌──────────────────────────────┐
│        Single Binary         │
│                              │
│  ┌────────────────────────┐  │
│  │   React + Vite + TS    │  │
│  │   (embedded via Go)    │  │
│  └────────┬───────────────┘  │
│           │                  │
│  ┌────────▼───────────────┐  │
│  │     Go HTTP Server     │  │
│  │  ┌──────┐ ┌──────────┐ │  │
│  │  │ SPA  │ │ API      │ │  │
│  │  │Fallbk│ │ Routes   │ │  │
│  │  └──────┘ └──────────┘ │  │
│  │  ┌──────┐ ┌──────────┐ │  │
│  │  │ Auth │ │WebSocket │ │  │
│  │  └──────┘ └──────────┘ │  │
│  └────────┬───────────────┘  │
│           │                  │
│  ┌────────▼───────────────┐  │
│  │   SQLite (pure Go)     │  │
│  │   No CGO required      │  │
│  └────────────────────────┘  │
│                              │
└──────────────────────────────┘
```

_"Yes, this was the plan all along."_ Me, lying.

## Things I Learned Along The Way

**1. SPA routing in embedded Go servers is a trap.**
If you serve a React app from Go and someone refreshes on `/shares`, Go goes "I don't have a file called `shares`, here's a 404." The fix: if the requested path has no file extension, serve `index.html` and let React Router figure it out. Simple in hindsight. Three hours of debugging in practice.

**2. CGO is the final boss of Go cross-compilation.**
`GOOS=linux GOARCH=arm64 go build` is beautiful until one of your dependencies whispers "I need C." Swapping `mattn/go-sqlite3` for `glebarez/sqlite` (which uses `modernc.org/sqlite`, a machine-translated C-to-Go SQLite) was the move. Zero CGO. Builds everywhere. The binary is bigger but my sanity is intact.

**3. `embed.FS` is criminally underrated.**
Go's `//go:embed` directive lets you bake your entire frontend build into the binary. No separate static file serving. No "make sure the `dist` folder is next to the executable." It just works. Ship one file. Run it. Done.

**4. You don't need a framework to build a framework.**
BeamDrop uses no web framework. No Gin, no Echo, no Fiber. Just `net/http` and a `ServeMux`. The standard library is enough. I'll die on this hill while manually parsing URL paths.

## Is It Actually a Framework Though?

Look, if it has:

- A backend server with routing and middleware ✅
- An embedded frontend with client-side routing ✅
- A database with ORM and migrations ✅
- Authentication (passwords, JWT, API keys) ✅
- Real-time communication (WebSockets) ✅
- An API layer (REST + S3-compatible) ✅
- Cross-platform single-binary builds ✅
- TLS support ✅
- CORS and security headers ✅

...then at what point does "my file sharing app" become "a full-stack framework with a file sharing app built on top of it"?

I don't have an answer. I just have a very over-engineered way to send a PDF to my phone.

## Try It

```bash
# Install (macOS Apple Silicon)
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-darwin-arm64.tar.gz | sudo tar -C /usr/local/bin -xzf -

# Run
beamdrop -dir ~/Documents -p secretpassword
```

That's it. One binary. Full-stack. No regrets. Okay, some regrets. But the QR code in the terminal is really cool.

---

_BeamDrop is open source at [github.com/ekilie/beamdrop](https://github.com/ekilie/beamdrop). Star it if you've ever accidentally over-engineered something. So, star it._
