
---

# 🏏 RaidX – Live Score Streaming System

## 📘 Overview

**RaidX** is a real-time sports scoring and tournament management platform designed to enable **live score updates**, **player management**, and **match tracking** with instant synchronization across all connected clients.

It’s built on a **Golang (Fiber)** backend and a **lightweight HTML/CSS/JS frontend**, using **MongoDB** for persistent data storage and **Redis** for fast, real-time caching.
The platform supports **multiple user roles** — admins, players, and viewers — each interacting with the system in a unique way.

RaidX is ideal for hosting tournaments (like kabaddi, cricket, etc.) where multiple teams compete and live score updates are broadcast instantly.

---

## 🎯 Objectives

* Provide a **real-time scoreboard** system for live tournaments.
* Support **multi-role access** — Admins, Players, and Viewers.
* Allow **admins** to manage teams, matches, and scoring.
* Deliver **instant score updates** using WebSockets without refreshing the page.
* Store historical match data securely in MongoDB.
* Cache active matches in Redis for maximum speed.

---

## 🧩 System Architecture

### 🏗️ High-Level Architecture

```
 ┌──────────────────────────────────────────────────┐
 │                    FRONTEND                      │
 │  Technologies: HTML, CSS, JavaScript             │
 │  - Views: home.html, login.html, signup.html     │
 │  - Real-time updates via WebSocket               │
 │  - REST API calls for player/team info           │
 └──────────────────────────────────────────────────┘
                      │
                      │ WebSocket + REST API
                      ▼
 ┌──────────────────────────────────────────────────┐
 │                    BACKEND                       │
 │  Built with Go + Fiber Framework                 │
 │                                                  │
 │  Modules:                                        │
 │   • internal/db → MongoDB connection setup        │
 │   • internal/redisImpl → Redis client setup       │
 │   • internal/handlers → API routes & logic        │
 │                                                  │
 │  Responsibilities:                               │
 │   • Handle team/player registration              │
 │   • Manage live scoring events                   │
 │   • Serve static pages                           │
 │   • Push updates via WebSocket                   │
 └──────────────────────────────────────────────────┘
                      │
         ┌────────────┴─────────────┐
         │                          │
 ┌───────▼────────┐        ┌────────▼───────┐
 │   MongoDB      │        │     Redis      │
 │ Persistent DB  │        │ In-memory Cache│
 │ - Players       │        │ - Live scores  │
 │ - Teams         │        │ - Ongoing data │
 │ - Matches       │        │ - Session info │
 └─────────────────┘        └────────────────┘
```

---

## ⚙️ Tech Stack

| Layer                 | Technology            | Purpose                                             |
| --------------------- | --------------------- | --------------------------------------------------- |
| **Frontend**          | HTML, CSS, JavaScript | User interface for players, admins, and viewers     |
| **Backend**           | Go (Fiber framework)  | REST API and WebSocket server                       |
| **Database**          | MongoDB               | Persistent data storage for players, teams, matches |
| **Cache Layer**       | Redis                 | Temporary caching of live scores for fast access    |
| **Templating Engine** | GoFiber HTML Engine   | Renders HTML templates dynamically                  |
| **Version Control**   | Git                   | Repository management and version tracking          |

---

## 🧱 Project Structure

```
RaidX-11-05/
│
├── main.go                     # Entry point of the application
├── go.mod                      # Go module dependencies
├── go.sum                      # Dependency verification hashes
│
├── internal/                   # Contains core backend logic
│   ├── db/                     # MongoDB initialization & operations
│   │   └── db.go               # Connection and query setup
│   │
│   ├── redisImpl/              # Redis configuration
│   │   └── redis.go            # Live score caching logic
│   │
│   └── handlers/               # API endpoints & handlers
│       ├── teamHandler.go      # Fetch teams, update scores, etc.
│       ├── matchHandler.go     # Manage live matches
│       └── userHandler.go      # Handle login, signup, etc.
│
├── Static/                     # Frontend static files
│   ├── home.html
│   ├── login.html
│   ├── signup.html
│   ├── scorer.html
│   └── startscore.html
│
└── views/                      # Go HTML templates (if used)
```

---

## 🧠 Core Features

### 👑 Admin

* Login and host tournaments.
* Create teams, assign players, and initiate matches.
* Manage match flow and push score updates in real time.
* Start or stop matches dynamically.

### 🧑‍🤝‍🧑 Players

* Register and log in to the platform.
* Get assigned to teams for tournaments.
* Participate in live matches.

### 👀 Viewers

* View live scores updated via WebSocket without refreshing.
* Access match details and team stats instantly.

---

## 🔌 API Endpoints

| Endpoint        | Method | Description                                          |
| --------------- | ------ | ---------------------------------------------------- |
| `/`             | GET    | Serve the homepage (`home.html`)                     |
| `/login`        | GET    | Serve the login page                                 |
| `/signup`       | GET    | Serve the signup page                                |
| `/scorer`       | GET    | Open the scoring interface                           |
| `/start`        | GET    | Start scoring session page                           |
| `/api/teams`    | GET    | Fetch all registered teams                           |
| `/api/team/:id` | GET    | Fetch specific team data by ID                       |
| `/ws/live`      | WS     | WebSocket endpoint for live updates (if implemented) |

---

## 🔁 Application Flow

1. **Initialization Phase**

   * The backend initializes MongoDB (`db.InitDB()`) and Redis (`redisImpl.InitRedis()`).
   * Fiber server is started and listens on the configured port.

2. **Frontend Routing**

   * Static HTML files (`home.html`, `login.html`, etc.) are served via Fiber routes.

3. **API & WebSocket Communication**

   * REST APIs handle data retrieval and updates (teams, players, matches).
   * WebSocket channels broadcast **live score updates** to all connected viewers.

4. **Real-Time Updates**

   * When the admin updates the score from `scorer.html`, the backend updates Redis.
   * Redis instantly pushes new values to all connected clients.
   * MongoDB stores the result for long-term tracking.

5. **End of Match**

   * Match results are finalized and stored permanently in MongoDB.
   * Redis clears live caches.

---

## 🧰 Installation & Setup

### Prerequisites

Ensure the following are installed:

* Go (v1.20 or newer)
* MongoDB (running on default port `27017`)
* Redis (running on default port `6379`)

### Steps

```bash
# Clone the repository
git clone https://github.com/yourusername/RaidX.git
cd RaidX-11-05

# Install dependencies
go mod tidy

# Run the Go Fiber app
go run main.go
```

By default, the server runs at:
👉 [http://localhost:3000](http://localhost:3000)

---

## 🧪 Example Workflow

1. Admin opens `http://localhost:3000/start` to create a new match.
2. Players register through `http://localhost:3000/signup`.
3. Scorer opens `/scorer` to manage scores.
4. Viewers open `/` to watch live updates in real time.
5. All data updates are reflected instantly through WebSocket + Redis.

---

## 📈 Future Enhancements

* Add **JWT authentication** for players/admins.
* Introduce **real-time commentary feed**.
* Build a **React/Next.js frontend** for better UX.
* Add **leaderboards** and **player analytics dashboard**.
* Enable **mobile notifications** for score changes.
* Add **Docker Compose** for containerized deployment (MongoDB + Redis + Go).

---

## 🧑‍💻 Developer Notes

* Use `.env` to manage MongoDB and Redis connection strings.
* Fiber HTML template engine is configured under `/views`.
* WebSocket integration can be expanded via `github.com/gofiber/websocket/v2`.

---

## 🏁 Conclusion

RaidX demonstrates a modern, lightweight, and efficient architecture for **real-time event-driven sports applications**.
It leverages the speed of **Go Fiber**, the scalability of **MongoDB**, and the power of **Redis caching** to deliver smooth and instantaneous score updates for live sports experiences.

---

