# Maze Go Runtime (`discordgo`)

This runtime owns:
- `/maze`
- `/maze-leaderboard`

It is designed to run **alongside** the existing JS bot during migration.

## Env
Uses the same `.env` values as JS:
- `DISCORD_TOKEN`
- `CLIENT_ID`
- `GUILD_ID` (optional, recommended for fast command updates)

## Run
1. In the JS bot environment, set:
   - `MAZE_RUNTIME=go`
2. Start JS bot (all non-maze commands):
   - `npm start`
3. Start Go maze runtime:
   - `npm run start:go-maze`

## Why this split
If both runtimes respond to the same command, Discord returns `already acknowledged`.
`MAZE_RUNTIME=go` makes JS skip maze commands so Go is the only responder for maze interactions.

