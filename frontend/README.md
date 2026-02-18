# RaidX Frontend (Phase 1)

This folder starts Phase 1 migration:

- `apps/web`: React website (login + role dashboard shell)
- `apps/mobile`: React Native app (login + role dashboard shell)
- `packages/shared`: shared auth/API client for both apps

## 1) Install dependencies

```bash
cd frontend
npm install
```

## 2) Run the Go backend

From project root (`RaidX-11-05`), run your backend so auth endpoints are available at `http://localhost:3000`.

## 3) Run web app

```bash
cd frontend
npm run dev:web
```

Web app uses Vite proxy to backend (`/login`, `/refresh`, `/logout`, `/logout-all`, `/api`).

## 4) Run mobile app

```bash
cd frontend
npm run dev:mobile
```

### Mobile backend URL

`apps/mobile/src/authClient.js` currently points to:

- `http://localhost:3000`

If you run on Android emulator, usually use `http://10.0.2.2:3000`.
If you run on physical device, use your machine LAN IP (e.g. `http://192.168.x.x:3000`).

## What is done in Phase 1 scaffold

- Auth flows: login, refresh, logout, logout-all
- Shared token/session management package
- Role-based dashboard shell route for web (`/dashboard/:role`)
- Role-based dashboard shell screen in mobile

## Next Phase 1 tasks

- Add role-specific dashboard data calls:
  - Player: `/api/player/teams`, `/api/player/events`
  - Owner: `/api/owner/teams`, `/api/owner/event-invitations`
  - Organizer: `/api/organizer/events`, `/api/organizer/event-invites`
- Add route guards per role
- Add profile and signup screens
