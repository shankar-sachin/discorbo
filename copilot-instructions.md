# Discorbo Go Bot - Developer Documentation

# Remember to build and run so I can test on Discord right after you finish coding

This document provides detailed technical documentation for the Discorbo Discord bot's Go implementation, including the economy system, tag game, and enhanced math capabilities.

**Last Updated:** 2026-02-18
**Bot Version:** 2.0.0 with Economy, Tag Game, and Enhanced Math

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Economy System](#economy-system)
3. [Tag Game System](#tag-game-system)
4. [Enhanced Math System](#enhanced-math-system)
5. [Data Persistence](#data-persistence)
6. [Button Interaction Patterns](#button-interaction-patterns)
7. [Session Management](#session-management)
8. [GDPR Compliance](#gdpr-compliance)

---

## Architecture Overview

### Technology Stack
- **Go 1.21+** - Main runtime
- **discordgo** - Discord API wrapper
- **JSON** - File-based data persistence
- **Mutex locks** - Thread-safe operations

### Key Files

| File | Purpose | Lines |
|------|---------|-------|
| `types.go` | All struct definitions, command declarations | ~450 |
| `cmd_economy.go` | Economy system handlers (shop, inventory, trade, admin) | ~900 |
| `cmd_fun_games.go` | Game commands including tag | ~700 |
| `helpers.go` | Math parser, embeds, tag game logic, unit conversion | ~900 |
| `handlers.go` | Button interaction routing | ~350 |
| `data.go` | Thread-safe JSON I/O | ~130 |
| `cmd_utility.go` | Utility commands including /convert | ~650 |

### Global State Variables

```go
var (
    botStartedAt   = time.Now()
    sessionMu      sync.Mutex           // Protects maze + tag sessions
    sessions       = map[string]*mazeSession{}
    dataMu         sync.Mutex           // Protects all JSON I/O
    triviaMu       sync.Mutex           // Protects trivia sessions
    triviaSessions = map[string]triviaSession{}
    tradeMu        sync.Mutex           // Protects trade sessions
    tradeSessions  = map[string]*tradeSession{}
    tagSessions    = map[string]*tagSession{}
)
```

**Thread Safety Rules:**
- All JSON reads/writes must acquire `dataMu`
- Trade sessions use `tradeMu` for in-memory modifications
- Tag and maze sessions share `sessionMu`
- Session maps are keyed by Discord message ID for automatic cleanup

---

## Economy System

### Overview

Full economy system with shops, inventory, player-to-player trading, boost mechanics, and admin controls. Integrates with existing coin rewards from daily claims, boss raids, and maze completion.

### Data Structures

#### Shop Item (`shopItem`)
```go
type shopItem struct {
    ID          string  `json:"id"`          // Unique identifier
    Name        string  `json:"name"`        // Display name
    Description string  `json:"description"` // What it does
    Price       int     `json:"price"`       // Cost in coins
    Emoji       string  `json:"emoji"`       // Visual indicator
    Category    string  `json:"category"`    // boost, collectible, consumable
    MaxOwned    int     `json:"maxOwned"`    // -1 = unlimited
    BoostType   string  `json:"boostType,omitempty"`   // daily_multiplier, raid_multiplier, etc.
    BoostValue  float64 `json:"boostValue,omitempty"`  // Multiplier value (2.0 = 2x)
}
```

#### Economy User (`economyUser`)
```go
type economyUser struct {
    Username     string          `json:"username"`
    Coins        int             `json:"coins"`        // Current balance
    Inventory    []inventoryItem `json:"inventory"`    // Owned items
    ActiveBoosts []activeBoost   `json:"activeBoosts"` // Currently active boosts
    TradeHistory []tradeRecord   `json:"tradeHistory"` // Past trades
    TotalSpent   int             `json:"totalSpent"`   // Lifetime spending
    TotalEarned  int             `json:"totalEarned"`  // Lifetime earnings
}
```

#### Active Boost (`activeBoost`)
```go
type activeBoost struct {
    BoostType  string  `json:"boostType"`  // Type of boost
    Multiplier float64 `json:"multiplier"` // Effect strength
    ExpiresAt  int64   `json:"expiresAt"`  // Unix timestamp (ms)
}
```

### Commands

#### `/shop list [category]`
- Displays paginated shop catalog
- Filter by: `all`, `boost`, `collectible`, `consumable`
- Sorted by price (ascending)
- Shows item emoji, name, price, max owned, description, ID

#### `/shop buy <item_id> [quantity]`
- Purchase items from catalog
- Validates:
  - Item exists
  - User has enough coins
  - Max owned limit not exceeded
- Deducts coins, adds to inventory
- Syncs coins to `daily-rewards.json`
- Logs transaction to `transactions.json`

#### `/shop sell <item_id> [quantity]`
- Sell items for 50% refund
- Removes from inventory
- Adds coins to balance
- Syncs to daily-rewards.json
- Logs transaction

#### `/inventory view`
- Shows all owned items with quantities
- Displays item emoji, name, quantity, and ID

#### `/inventory use <item_id>`
- Activates consumable/boost items
- **Boosts**: Adds to `activeBoosts` array with expiration
  - If boost already active, extends duration
  - Duration varies: 24h (daily), 12h (xp), 6h (global coins)
- **Mystery Box**: Opens for random coin reward (50-500 coins)
- Decrements quantity, removes if zero

#### `/inventory info <item_id>`
- Detailed item inspection
- Shows: price, category, owned quantity, max owned, boost effect

#### `/balance`
- Shows current coins, total earned, total spent
- Lists active boosts with expiration times
- Displays inventory item count
- Auto-cleans expired boosts

#### `/trade offer <user>`
- Initiates trade session (placeholder)
- Currently shows "under development" message
- Session structure ready for full implementation
- Auto-expires after 5 minutes

#### `/economy-admin grant <user> <coins> [reason]`
- **Admin only** (requires `Administrator` permission)
- Grants coins to user
- Updates both `economy-users.json` and `daily-rewards.json`
- Logs to transactions with admin ID and reason

#### `/economy-admin take <user> <coins> [reason]`
- **Admin only**
- Removes coins from user (cannot go negative)
- Updates both economy and daily files
- Logs transaction

#### `/economy-admin transactions [user]`
- **Admin only**
- Views last 10 transactions
- Filter by user or show all
- Displays: type, amount, reason, timestamp

### JSON Files

#### `shop-catalog.json`
Admin-editable shop inventory. Example items:
```json
{
  "daily_boost": {
    "id": "daily_boost",
    "name": "Daily Boost Potion",
    "description": "2x daily rewards for 24 hours",
    "price": 500,
    "emoji": "🧪",
    "category": "boost",
    "maxOwned": 5,
    "boostType": "daily_multiplier",
    "boostValue": 2.0
  },
  "lucky_coin": {
    "id": "lucky_coin",
    "name": "Lucky Coin",
    "description": "Rare collectible golden coin",
    "price": 1000,
    "emoji": "🪙",
    "category": "collectible",
    "maxOwned": -1
  }
}
```

#### `economy-users.json`
User balances and inventories:
```json
{
  "userID": {
    "username": "Player1",
    "coins": 2500,
    "inventory": [
      {"itemId": "daily_boost", "quantity": 2, "purchasedAt": 1234567890}
    ],
    "activeBoosts": [
      {"boostType": "daily_multiplier", "multiplier": 2.0, "expiresAt": 1234567899}
    ],
    "tradeHistory": [],
    "totalSpent": 1500,
    "totalEarned": 4000
  }
}
```

#### `transactions.json`
Audit log (last 1000 transactions):
```json
[
  {
    "userId": "userID",
    "type": "purchase",
    "itemId": "daily_boost",
    "amount": -500,
    "timestamp": 1234567890
  },
  {
    "userId": "userID",
    "type": "admin_grant",
    "amount": 1000,
    "grantedBy": "adminID",
    "reason": "Event winner",
    "timestamp": 1234567891
  }
]
```

### Boost Integration

**In `handleDaily()` (cmd_fun_games.go:132-142):**
```go
// Apply active boosts
boosts := getActiveBoosts(user.ID)
multiplier := 1.0
for _, b := range boosts {
    if b.BoostType == "daily_multiplier" || b.BoostType == "global_coin_multiplier" {
        multiplier *= b.Multiplier
    }
}
if multiplier > 1.0 {
    reward = int(float64(reward) * multiplier)
}
```

**In `handleBossRaid()` attack case (cmd_fun_games.go:247-257):**
```go
// Apply active boosts
boosts := getActiveBoosts(user.ID)
multiplier := 1.0
for _, b := range boosts {
    if b.BoostType == "raid_multiplier" || b.BoostType == "global_coin_multiplier" {
        multiplier *= b.Multiplier
    }
}
if multiplier > 1.0 {
    damage = int(float64(damage) * multiplier)
}
```

### Coin Synchronization

**Two-way sync between `economy-users.json` and `daily-rewards.json`:**
- **On purchase/sell**: Updates economy file, then syncs coins to daily-rewards
- **On balance check**: Reads daily-rewards as source of truth
- **Why**: Daily rewards is the original coin system; economy extends it

### Helper Functions

#### `getActiveBoosts(userID string) []activeBoost`
- Reads `economy-users.json`
- Filters expired boosts (ExpiresAt < now)
- Returns only active boosts
- Called by daily/raid handlers

#### `logTransaction(userID, txType, itemID string, amount int, grantedBy, reason string)`
- Appends to `transactions.json`
- Keeps last 1000 transactions (auto-trims)
- Transaction types: `purchase`, `sell`, `admin_grant`, `admin_take`, `trade`

---

## Tag Game System

### Overview

Turn-based strategic tag game on a 5x5 grid with obstacles. Players take turns moving up/down/left/right/stay, trying to occupy the same position as their opponent. Rewards coins based on speed of victory.

### Data Structures

#### Tag Session (`tagSession`)
```go
type tagSession struct {
    SessionID   string
    Player1ID   string
    Player1Name string
    Player2ID   string
    Player2Name string
    CurrentTurn int        // 0 = P1, 1 = P2
    Player1X    int        // Player 1 X position (0-4)
    Player1Y    int        // Player 1 Y position (0-4)
    Player2X    int        // Player 2 X position (0-4)
    Player2Y    int        // Player 2 Y position (0-4)
    Board       [][]string // 5x5 grid with obstacles
    Moves       int        // Current move count
    MaxMoves    int        // 20 moves maximum
    StartTime   int64      // Unix timestamp
    MessageID   string     // Discord message ID
    ChannelID   string     // Discord channel ID
}
```

#### Tag Stats (`tagStats`)
```json
{
  "username": "Player1",
  "wins": 15,
  "losses": 8,
  "totalGames": 23,
  "coinsEarned": 750
}
```

### Commands

#### `/tag challenge <opponent>`
- Initiates tag game with another user
- Sends accept/decline buttons to opponent
- Creates in-memory session
- Auto-expires after 5 minutes if not accepted

#### `/tag leaderboard`
- Shows top 10 players by wins
- Displays: wins, losses, total games, coins earned
- Sorted descending by wins

#### `/tag stats [user]`
- View tag statistics for self or another user
- Shows: wins, losses, win rate %, total games, coins earned

### Game Board

**5x5 Grid:**
```
[P1] [ ] [#] [ ] [ ]
[ ] [#] [ ] [ ] [#]
[ ] [ ] [#] [ ] [ ]
[#] [ ] [ ] [#] [ ]
[ ] [ ] [#] [ ] [P2]
```

- `🔵` = Player 1 (starts at 0,0)
- `🔴` = Player 2 (starts at 4,4)
- `⬛` = Wall/Obstacle (8 randomly placed, avoid spawn points)
- `⬜` = Empty space

### Game Flow

1. **Challenge**: `/tag challenge @opponent`
2. **Accept**: Opponent clicks Accept button
3. **Board Generation**: `generateTagBoard()` creates random 5x5 grid
4. **Turn Loop**:
   - Current player sees directional buttons (⬆️⬇️⬅️➡️ + Stay)
   - Click button to move
   - Movement validated (bounds, walls)
   - Check win condition (same position)
   - Switch turn
   - Increment move counter
5. **Win Condition**: Players occupy same position → Winner declared
6. **Draw Condition**: 20 moves reached → Both get 10 coins
7. **Rewards**:
   - Winner: `50 + (remainingMoves * 10)` coins
   - Example: Win on move 5 → 50 + (15 * 10) = 200 coins
   - Draw: Both players get 10 coins

### Board Generation Algorithm

**`generateTagBoard()` in helpers.go:**
```go
func generateTagBoard() [][]string {
    board := make([][]string, 5)
    for i := range board {
        board[i] = make([]string, 5)
        for j := range board[i] {
            board[i][j] = "⬜"
        }
    }

    // Place 8 random obstacles, avoid (0,0) and (4,4)
    obstacles := 0
    for obstacles < 8 {
        x, y := rand.Intn(5), rand.Intn(5)
        if (x == 0 && y == 0) || (x == 4 && y == 4) {
            continue
        }
        if board[y][x] == "⬜" {
            board[y][x] = "⬛"
            obstacles++
        }
    }

    return board
}
```

### Move Validation

**`processTagMove(sess *tagSession, direction string) bool`:**
1. Get current player's coordinates
2. Calculate new position based on direction
3. Check bounds (0-4)
4. Check for walls (`⬛`)
5. Apply move if valid
6. Return true/false

### Button Interactions

**CustomID patterns:**
- `tag_accept_{sessionID}` - Accept challenge
- `tag_decline_{sessionID}` - Decline challenge
- `tag_up_{sessionID}` - Move up
- `tag_down_{sessionID}` - Move down
- `tag_left_{sessionID}` - Move left
- `tag_right_{sessionID}` - Move right
- `tag_stay_{sessionID}` - Stay in place

**Handler: `handleTagComponent()` in handlers.go:**
- Validates current player's turn
- Processes move
- Updates board
- Checks win/draw
- Awards coins and updates stats
- Deletes session on game end

### Session Management

- **Storage**: `tagSessions = map[string]*tagSession{}` (keyed by message ID)
- **Mutex**: Uses `sessionMu` (shared with maze sessions)
- **Cleanup**: Auto-delete after 5 minutes if game not started
- **Game End**: Immediate deletion, prevents replay

### Stats Tracking

**Updated on game end:**
- Winner: `wins++`, `totalGames++`, `coinsEarned += amount`
- Loser: `losses++`, `totalGames++`
- Stored in `tag-stats.json`
- Synced to `daily-rewards.json` for coins

---

## Enhanced Math System

### Overview

Upgraded calculator with support for advanced functions, mathematical constants, scientific notation, and unit conversions.

### Math Constants

```go
const (
    mathPi  = 3.141592653589793  // π
    mathE   = 2.718281828459045  // Euler's number
    mathTau = 6.283185307179586  // τ (2π)
    mathPhi = 1.618033988749895  // Golden ratio
)
```

**Usage in expressions:**
- `2 * pi` → 6.283185...
- `e^2` → 7.389056...
- `tau / 2` → 3.141592... (same as pi)
- `phi^2` → 2.618033...

### Math Functions

**Supported functions (all take single argument):**

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `sqrt(x)` | Square root | `sqrt(16)` | 4 |
| `sin(x)` | Sine (radians) | `sin(pi/2)` | 1 |
| `cos(x)` | Cosine (radians) | `cos(0)` | 1 |
| `tan(x)` | Tangent (radians) | `tan(pi/4)` | 1 |
| `asin(x)` | Arcsine | `asin(1)` | 1.5707... |
| `acos(x)` | Arccosine | `acos(0)` | 1.5707... |
| `atan(x)` | Arctangent | `atan(1)` | 0.7853... |
| `log(x)` | Log base 10 | `log(100)` | 2 |
| `ln(x)` | Natural log | `ln(e)` | 1 |
| `abs(x)` | Absolute value | `abs(-5)` | 5 |
| `ceil(x)` | Ceiling | `ceil(4.2)` | 5 |
| `floor(x)` | Floor | `floor(4.8)` | 4 |
| `round(x)` | Round | `round(4.5)` | 5 |
| `exp(x)` | e^x | `exp(1)` | 2.7182... |

### Scientific Notation

**Supported formats:**
- `1e6` → 1000000
- `2.5e3` → 2500
- `5e-2` → 0.05
- `1.23e-4` → 0.000123

**Parsed by regex:** `(\d+(?:\.\d+)?)e([+-]?\d+)`

### Expression Parser Flow

**`evalMath(expr string)` pipeline:**
1. **Replace constants**: `pi` → `3.141592`, `e` → `2.718281`, etc.
2. **Parse scientific notation**: `1e6` → `1000000`
3. **Evaluate functions**: `sqrt(16)` → `4`, recursively for nested functions
4. **Recursive descent parser**: Handles `+`, `-`, `*`, `/`, `^`, `%`, parentheses
5. **Return result**: Integer if whole number, else float

**Function evaluation (recursive):**
```go
func evaluateFunctions(expr string) (string, error) {
    funcPattern := regexp.MustCompile(`(\w+)\(([^)]+)\)`)

    for funcPattern.MatchString(expr) {
        matches := funcPattern.FindStringSubmatch(expr)
        funcName := matches[1]
        argExpr := matches[2]

        // Recursively evaluate argument (supports nested functions)
        argResult, err := evalMath(argExpr)
        argVal, _ := strconv.ParseFloat(argResult, 64)
        result, err := parseFunction(funcName, argVal)

        // Replace function call with result
        expr = strings.Replace(expr, matches[0], fmt.Sprintf("%f", result), 1)
    }

    return expr, nil
}
```

**Nested function example:**
- `sqrt(abs(-16))`
- Step 1: `abs(-16)` → `16`
- Step 2: `sqrt(16)` → `4`

### `/calc` Command

**Usage:**
```
/calc expression: 2*pi
→ Result: 6.283185307179586

/calc expression: sqrt(16) + log(100)
→ Result: 6

/calc expression: sin(pi/2) * 100
→ Result: 100

/calc expression: 1.5e6 / 1000
→ Result: 1500
```

**Error handling:**
- Invalid syntax: Shows supported features
- Negative sqrt/log: Error message
- Unknown function: Lists available functions

### `/convert` Command

**Temperature conversions:**
```
/convert value:32 from:f to:c
→ 32.00°F = 0.00°C

/convert value:100 from:c to:f
→ 100.00°C = 212.00°F

/convert value:273.15 from:k to:c
→ 273.15°K = 0.00°C
```

**Length conversions:**
```
/convert value:5 from:mi to:km
→ 5.00 mi = 8.05 km

/convert value:100 from:ft to:m
→ 100.00 ft = 30.48 m

/convert value:10 from:in to:cm
→ 10.00 in = 25.40 cm
```

**Weight conversions:**
```
/convert value:150 from:lb to:kg
→ 150.00 lb = 68.04 kg

/convert value:500 from:g to:oz
→ 500.00 g = 17.64 oz
```

**Volume conversions:**
```
/convert value:1 from:gal to:l
→ 1.00 gal = 3.79 l

/convert value:250 from:ml to:cup
→ 250.00 ml = 1.06 cup
```

### Unit Conversion Implementation

**Temperature (special case with offsets):**
```go
func convertTemperature(value float64, from, to string) string {
    var celsius float64

    // Convert to Celsius first
    switch from {
    case "f": celsius = (value - 32) * 5 / 9
    case "k": celsius = value - 273.15
    case "c": celsius = value
    }

    // Convert from Celsius to target
    switch to {
    case "f": result = celsius*9/5 + 32
    case "k": result = celsius + 273.15
    case "c": result = celsius
    }

    return fmt.Sprintf("%.2f°%s = %.2f°%s", value, from, result, to)
}
```

**Other units (multiplier-based):**
```go
conversions := map[string]float64{
    "ft_to_m":  0.3048,
    "m_to_ft":  3.28084,
    "mi_to_km": 1.60934,
    "km_to_mi": 0.621371,
    "in_to_cm": 2.54,
    "cm_to_in": 0.393701,
    "lb_to_kg": 0.453592,
    "kg_to_lb": 2.20462,
    "oz_to_g":  28.3495,
    "g_to_oz":  0.035274,
    "gal_to_l": 3.78541,
    "l_to_gal": 0.264172,
}
```

### Enhanced Dice Rolling

**`/roll` improvements:**

**Features:**
- Shows individual rolls in array: `[5, 12, 19]`
- Calculates and displays average roll
- Highlights natural 20s with `**bold**`
- Highlights natural 1s with `*italic*`
- Displays modifier separately
- Shows total in embed field

**Example output:**
```
🎲 Rolling 3d20+5

Rolls: [12, **20**, 8]
Average: 13.3
Modifier: +5

🎉 NATURAL 20! Critical success!

Total: 45
```

**Detection logic:**
```go
hasNat20 := false
hasNat1 := false

for _, r := range rolls {
    if sides == 20 && r == 20 {
        hasNat20 = true
    }
    if r == 1 {
        hasNat1 = true
    }
}
```

---

## Data Persistence

### JSON File Structure

**All JSON files stored in:** `../src/data/` (relative to go/ directory)

| File | Type | Purpose | Schema |
|------|------|---------|--------|
| `shop-catalog.json` | Object | Shop items | `{itemID: shopItem}` |
| `economy-users.json` | Object | User economy data | `{userID: economyUser}` |
| `transactions.json` | Array | Transaction log | `[transactionLog]` |
| `tag-stats.json` | Object | Tag game stats | `{userID: tagStats}` |
| `daily-rewards.json` | Object | Coin balances | `{userID: dailyUser}` |
| `trivia-scores.json` | Object | Trivia scores | `{userID: triviaScore}` |
| `loot.json` | Object | Loot inventories | `{userID: lootUser}` |
| `quests.json` | Object | Quest progress | `{userID: questUser}` |
| `battle-stats.json` | Object | Battle records | `{userID: battleStats}` |
| `afk-users.json` | Object | AFK statuses | `{userID: afkStatus}` |
| `maze-leaderboard.json` | Object | Maze stats | `{userID: userMazeData}` |
| `reminders.json` | Array | Active reminders | `[reminderEntry]` |
| `quotes.json` | Object | Server quotes | `{guildID: quoteGuild}` |
| `bossraid.json` | Object | Boss raid state | `{guildID: raidGuild}` |

### Thread-Safe I/O Functions

**`readData(file string, out any) error`:**
- Acquires `dataMu` mutex
- Creates file with `{}` if doesn't exist
- Reads and unmarshals JSON
- Returns error or nil

**`writeData(file string, in any) error`:**
- Acquires `dataMu` mutex
- Marshals to indented JSON
- Atomically writes to file
- Returns error or nil

**`clearUserKeyInObject(file, userID string)`:**
- Reads object from file
- Deletes user's key
- Writes back
- Used by GDPR cleanup

### Data Migration Notes

**Coin synchronization:**
- `daily-rewards.json` is source of truth for coins
- `economy-users.json` maintains extended economy data
- On shop purchase/sell: Update economy → sync to daily
- On balance check: Read from daily → display in economy
- This prevents coin duplication and maintains backward compatibility

---

## Button Interaction Patterns

### CustomID Naming Convention

**Format:** `{system}_{action}_{identifier}[_{extra}]`

**Examples:**
- `trivia_answer_token123_2` - Trivia answer button 2 for session token123
- `maze_up` - Maze move up button
- `trade_accept_sessionID` - Accept trade offer
- `tag_up_sessionID` - Tag game move up
- `tag_decline_sessionID` - Decline tag challenge

### Handler Routing

**In `handlers.go` → `handleComponent()`:**
```go
func handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
    data := i.MessageComponentData()

    if strings.HasPrefix(data.CustomID, "trivia_") {
        handleTriviaComponent(s, i)
        return
    }
    if strings.HasPrefix(data.CustomID, "trade_") {
        handleTradeComponent(s, i)
        return
    }
    if strings.HasPrefix(data.CustomID, "tag_") {
        handleTagComponent(s, i)
        return
    }
    if strings.HasPrefix(data.CustomID, "maze_") {
        handleMazeComponent(s, i)
        return
    }
    // Default: ignore unknown buttons
}
```

### Button Creation Pattern

**Action Row with buttons:**
```go
buttons := discordgo.ActionsRow{
    Components: []discordgo.MessageComponent{
        discordgo.Button{
            Label:    "Accept",
            Style:    discordgo.SuccessButton,  // Green
            CustomID: fmt.Sprintf("tag_accept_%s", sessionID),
        },
        discordgo.Button{
            Label:    "Decline",
            Style:    discordgo.DangerButton,   // Red
            CustomID: fmt.Sprintf("tag_decline_%s", sessionID),
        },
    },
}

// Send with message
resp := &discordgo.InteractionResponse{
    Type: discordgo.InteractionResponseChannelMessageWithSource,
    Data: &discordgo.InteractionResponseData{
        Embeds:     []*discordgo.MessageEmbed{embed},
        Components: []discordgo.MessageComponent{buttons},
    },
}
```

### Button Styles

| Style | Color | Use Case |
|-------|-------|----------|
| `PrimaryButton` | Blue | Navigation, neutral actions |
| `SuccessButton` | Green | Confirm, accept, positive actions |
| `DangerButton` | Red | Cancel, decline, destructive actions |
| `SecondaryButton` | Gray | Alternative options |

### Session Tracking via Message ID

**Pattern:**
1. Send interaction response with buttons
2. Get message ID from response: `msg, _ := s.InteractionResponse(i.Interaction)`
3. Store session in map: `sessions[msg.ID] = sessionData`
4. On button click: Lookup session by `i.Message.ID`
5. On game end: Delete session from map

**Benefits:**
- No database needed
- Automatic cleanup when message deleted
- Fast lookups (O(1) map access)
- Message-specific state isolation

---

## Session Management

### In-Memory Session Storage

**Three session maps:**
```go
sessions       = map[string]*mazeSession{}   // Maze games
tagSessions    = map[string]*tagSession{}    // Tag games
tradeSessions  = map[string]*tradeSession{}  // Trade offers
triviaSessions = map[string]triviaSession{}  // Trivia questions
```

### Session Lifecycle

**Creation:**
1. User triggers command (`/tag challenge`, `/trade offer`, etc.)
2. Generate unique session ID: `fmt.Sprintf("%s_%s_%d", user1, user2, timestamp)`
3. Send Discord message with buttons
4. Store session with message ID as key
5. Launch auto-cleanup goroutine

**Interaction:**
1. User clicks button
2. Handler extracts `i.Message.ID`
3. Lookup session in map with mutex
4. Validate user permission
5. Process action
6. Update embed
7. Respond with updated message

**Cleanup:**
- **Automatic**: Goroutine deletes after timeout (5 minutes)
- **Manual**: Delete on game end (win/loss/draw)
- **Expired**: Filter by ExpiresAt timestamp

**Example cleanup goroutine:**
```go
go func(msgID string) {
    time.Sleep(5 * time.Minute)
    sessionMu.Lock()
    delete(tagSessions, msgID)
    sessionMu.Unlock()
}(msg.ID)
```

### Mutex Strategy

**`sessionMu`** - Shared by maze and tag games
- Acquire before reading/writing session maps
- Release immediately after operation
- Use defer for safety

**`tradeMu`** - Trade sessions only
- Prevents race conditions in offer modifications
- Critical for atomic coin/item transfers

**`dataMu`** - All JSON I/O
- Most frequently acquired mutex
- Held during entire read/write operation
- Prevents file corruption from concurrent writes

**Best Practices:**
- Keep mutex-locked sections minimal
- Never call blocking operations inside mutex
- Use defer for unlock: `defer mu.Unlock()`
- Acquire in consistent order to prevent deadlocks

---

## GDPR Compliance

### `/clear-my-data` Command

**Full data deletion for user:**

**Files cleared (object-based):**
- `clearUserKeyInObject("trivia-scores.json", user.ID)`
- `clearUserKeyInObject("daily-rewards.json", user.ID)`
- `clearUserKeyInObject("loot.json", user.ID)`
- `clearUserKeyInObject("quests.json", user.ID)`
- `clearUserKeyInObject("battle-stats.json", user.ID)`
- `clearUserKeyInObject("afk-users.json", user.ID)`
- `clearUserKeyInObject("maze-leaderboard.json", user.ID)`
- `clearUserKeyInObject("economy-users.json", user.ID)` ← New
- `clearUserKeyInObject("tag-stats.json", user.ID)` ← New

**Array-based cleanup:**
```go
// Reminders
reminders := readReminders()
kept := []reminderEntry{}
for _, r := range reminders {
    if r.UserID != user.ID {
        kept = append(kept, r)
    }
}
writeReminders(kept)

// Transactions
transactions := []transactionLog{}
readData("transactions.json", &transactions)
keptTx := []transactionLog{}
for _, t := range transactions {
    if t.UserID != user.ID {
        keptTx = append(keptTx, t)
    }
}
writeData("transactions.json", keptTx)
```

**Note:** Guild-scoped data (quotes, boss raids) is not deleted as it's shared data.

### Data Retention Policy

- **User-specific data**: Deleted on request
- **Guild-specific data**: Retained (shared resource)
- **Transactions**: User's transactions removed from log
- **Trade history**: Removed from user's economy profile

### Privacy Considerations

- No personally identifiable information stored beyond Discord IDs
- Usernames stored for display only (not for identification)
- All data stored locally (no third-party services)
- No message content logged (except quotes, which are intentional)

---

## Development Workflow

### Adding New Economy Items

1. Edit `src/data/shop-catalog.json`
2. Add new item with unique ID
3. Set category, price, boost type/value
4. Restart bot (no code changes needed)

### Adding New Math Functions

1. Add to `parseFunction()` in helpers.go
2. Use Go's `math` package for calculation
3. Add error handling for invalid inputs
4. Update calc error message with new function
5. Test with `/calc`

### Adding New Unit Conversions

1. Add to `conversions` map in `convertUnits()` (cmd_utility.go)
2. Format: `"{from}_to_{to}": multiplier`
3. For temperature, extend `convertTemperature()` switch cases
4. Test with `/convert`

### Testing Checklist

**Economy System:**
- [ ] `/shop list` displays all items
- [ ] `/shop buy` deducts coins correctly
- [ ] `/shop sell` refunds 50%
- [ ] `/inventory use` activates boosts
- [ ] Boosts apply to daily/raid rewards
- [ ] `/balance` shows active boosts
- [ ] `/economy-admin grant` works for admins only
- [ ] GDPR cleanup removes economy data

**Tag Game:**
- [ ] Challenge sends buttons to opponent
- [ ] Declining cancels cleanly
- [ ] Board generates with 8 obstacles
- [ ] Moves validate walls and bounds
- [ ] Same position triggers win
- [ ] 20 moves triggers draw
- [ ] Coins awarded correctly
- [ ] Stats update properly

**Enhanced Math:**
- [ ] Constants work: `2*pi`, `e^2`
- [ ] Functions work: `sqrt(16)`, `sin(pi/2)`
- [ ] Nested functions: `sqrt(abs(-16))`
- [ ] Scientific notation: `1e6`, `2.5e-3`
- [ ] `/convert` handles all unit types
- [ ] Dice roll shows averages and crits

---

## Performance Optimization

### Mutex Granularity

- **Current**: One mutex per subsystem (session, data, trade, trivia)
- **Benefit**: Allows concurrent operations across subsystems
- **Trade-off**: More complex locking logic

### JSON File Size Management

- **Transactions**: Auto-trim to last 1000 entries
- **Reminders**: Cleaned up on delivery
- **Sessions**: In-memory only, no persistence

### Memory Footprint

- **Sessions**: Auto-deleted after timeout/completion
- **Boosts**: Filtered on read (expired boosts dropped)
- **Leaderboards**: Cached during read, not kept in memory

### Scalability Considerations

**Current limits:**
- ~100 concurrent tag/trade sessions
- ~1000 shop items
- ~10,000 users per economy file
- ~1MB JSON files load in <10ms

**Migration path for large bots:**
- Replace JSON with PostgreSQL/MongoDB
- Add Redis for session caching
- Implement sharding for 2500+ servers
- Move transactions to separate database

---

## Error Handling

### User-Facing Errors

**Always show:**
- What went wrong (validation failure, insufficient funds, etc.)
- What the user can do to fix it
- Never show stack traces or internal errors

**Example:**
```go
if ecoUser.Coins < totalCost {
    respondText(s, i, fmt.Sprintf("Not enough coins. Need %d, have %d.", totalCost, ecoUser.Coins))
    return
}
```

### Logging

**Console output:**
- Command usage: `logger.command()`
- Errors: `logger.error()`
- Info: `logger.info()`

**No logging to files** (reduces complexity)

### Graceful Degradation

- If shop catalog fails to load: Show error, don't crash
- If session not found: Inform user, don't panic
- If JSON parse fails: Use default empty object/array

---

## Common Patterns

### Option Extraction

```go
sub, subOpts := getSubcommand(opts)
itemID := optionString(subOpts, "item_id", "")
quantity := int(optionInt(subOpts, "quantity", 1))
targetUser := optionUser(subOpts, "user")
```

### Embed Response

```go
embed := createSuccessEmbed("Title", "Description")
embed.Fields = []*discordgo.MessageEmbedField{
    {Name: "Field", Value: "Value", Inline: true},
}
respondEmbed(s, i, embed)
```

### Data Read/Write

```go
data := map[string]someType{}
_ = readData("file.json", &data)

// Modify data

_ = writeData("file.json", data)
```

### Session Pattern

```go
sessionMu.Lock()
sess, ok := sessions[i.Message.ID]
if !ok {
    sessionMu.Unlock()
    // Error response
    return
}

// Modify session

sessionMu.Unlock()

// Respond to user
```

---

## Future Enhancements

### Economy System
- Full trade UI with coin/item selection
- Auction house system
- Daily shop rotation
- Seasonal items
- Crafting system

### Tag Game
- Different board sizes (3x3, 7x7)
- Power-ups (teleport, freeze opponent)
- Time-based mode (speed chess style)
- Team tag (2v2)

### Math System
- Matrix operations
- Complex numbers
- Graphing support
- Equation solver

### Infrastructure
- Database migration (PostgreSQL)
- Redis caching layer
- Prometheus metrics
- Rate limiting per user
- Command usage analytics dashboard

---

**End of Documentation**

For questions or contributions, refer to the main CLAUDE.md or create an issue in the repository.
