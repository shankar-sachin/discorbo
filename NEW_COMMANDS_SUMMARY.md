# 🎮 NEW COMMANDS ADDED TO DISCORBO

## Overview
Added **11 new epic commands** to transform Discorbo into an even more engaging Discord bot with RPG elements, social features, and competitive gameplay!

---

## 🎯 Core Fun Commands (5 Commands)

### `/vibecheck [user]`
**Description:** Check your or someone else's vibe rating for the day
**Features:**
- 10+ unique vibe ratings (Immaculate, Main Character Energy, Villain Arc, etc.)
- Deterministic daily vibes (same result for the day)
- Animated vibe meter with percentage
- Custom colors per vibe type

**Example Output:**
```
✨ Vibe Check: Username
Status: Immaculate ✨
Your vibe is absolutely *chef's kiss*. The energy is unmatched.

Vibe Meter:
████████████████████ 100%
```

---

### `/quest`
**Description:** RPG-style quest system with XP and leaderboards
**Subcommands:**
- `/quest get` - Get a new random quest
- `/quest complete` - Mark quest as complete and earn XP
- `/quest leaderboard` - View top questers
- `/quest stats` - View your quest statistics

**Features:**
- 20+ hilarious quest templates
- Difficulty tiers: Easy, Medium, Hard, Legendary
- XP rewards based on difficulty
- Leveling system (100 XP per level)
- Quest leaderboard with XP tracking

**Example Quests:**
- "Retrieve the sacred pizza from the Forbidden Fridge" (Easy - 50 XP)
- "Complete a group project where everyone actually contributes" (Legendary - 500 XP)
- "Achieve inbox zero" (Legendary - 300 XP)

---

### `/roll <dice>`
**Description:** Advanced dice rolling with dramatic flair
**Features:**
- Supports standard notation: d20, 3d6, 2d10+5
- Critical hit detection (max roll)
- Critical failure detection (nat 1)
- Dramatic flavor text
- Visual result presentation

**Example Usage:**
```
/roll d20          → Roll a d20
/roll 3d6          → Roll three d6 dice
/roll 2d10+5       → Roll 2d10 and add 5
/roll 1d20-2       → Roll d20 and subtract 2
```

---

### `/battle @user`
**Description:** Turn-based combat system with stats tracking
**Features:**
- Turn-based battle system
- 4 unique attack types with different damage ranges
- Critical hit mechanics (10-25% chance depending on attack)
- HP tracking with visual health bars
- Battle statistics (wins, losses, win streaks)
- Win streak tracking and display

**Attack Types:**
- Quick Jab: 8-15 damage (10% crit)
- Power Strike: 15-25 damage (15% crit)
- Tactical Shot: 10-20 damage (20% crit)
- Chaos Blast: 5-35 damage (25% crit)

---

### `/loot`
**Description:** Loot box system with inventory
**Subcommands:**
- `/loot open` - Open a loot chest
- `/loot inventory` - View your items
- `/loot stats` - View loot statistics

**Rarity Tiers:**
- ⚪ Common (35%) - "Slightly Used Tissue"
- 🟢 Uncommon (25%) - "Working Charger"
- 🔵 Rare (20%) - "Full Night's Sleep"
- 🟣 Epic (12%) - "WiFi That Actually Works"
- 🟡 Legendary (6%) - "Unlimited Garlic Bread"
- 🌌 Cosmic (1.5%) - "The Ability to Pause Time"
- 💀 Cursed (0.5%) - "Wet Socks (Permanent)"

---

## 🌟 Social/Chaos Commands (3 Commands)

### `/hotseat`
**Description:** Put a random server member in the spotlight
**Features:**
- Randomly selects a server member
- 30+ spicy/funny questions
- Public call-out with ping

**Example Questions:**
- "What's your most embarrassing Discord moment?"
- "What's the weirdest thing in your search history right now?"
- "Be honest: How often do you leave people on read?"

---

### `/quote`
**Description:** Save and retrieve iconic server quotes
**Subcommands:**
- `/quote add <text> [author]` - Save a quote
- `/quote random` - Get a random quote
- `/quote list` - List recent quotes
- `/quote remove <id>` - Remove quote (moderators only)

---

### `/summon @user [reason]`
**Description:** Dramatically summon someone to the conversation
**Features:**
- 15+ unique summoning styles
- Fantasy/RPG-themed messages
- Optional reason parameter

**Summon Types:**
- ⚡ Divine Summons
- 🔮 Mystic Summoning
- 👑 Royal Decree
- 💀 Necromantic Ritual
- 🐉 Dragon's Roar

---

## 💪 Power-User Commands (2 Commands)

### `/bossraid`
**Description:** Server-wide cooperative boss battles
**Subcommands:**
- `/bossraid status` - Check current boss
- `/bossraid attack` - Attack the boss
- `/bossraid spawn` - Spawn new boss (Admin only)
- `/bossraid leaderboard` - View damage leaderboard

**Features:**
- 6 unique bosses with themed loot
- 5-minute attack cooldown per user
- Critical hit system (15% chance)
- Damage leaderboard
- Coin rewards on boss defeat

**Boss Examples:**
- 👾 The Lag Monster (5000 HP)
- 😈 The Procrastination Demon (6000 HP)
- 📅 The Monday Overlord (7000 HP)

---

### `/daily`
**Description:** Daily rewards and streak system
**Subcommands:**
- `/daily claim` - Claim daily reward
- `/daily stats` - View your stats
- `/daily leaderboard` - View coins leaderboard

**Features:**
- 100 base coins per day
- Streak bonuses (10 coins per day, max +500)
- 10% chance for 2x lucky bonus
- Leaderboard with streak indicators

**Reward Calculation:**
```
Base: 100 coins
Streak Bonus: Current Streak × 10 (max 500)
Lucky Bonus: 10% chance for 2x multiplier
```

---

## 📊 New Data Files

1. **quests.json** - Quest progress and XP
2. **battle-stats.json** - Battle records and win streaks
3. **loot.json** - Item inventories and loot stats
4. **quotes.json** - Server quotes (per-guild)
5. **bossraid.json** - Active bosses and participants
6. **daily-rewards.json** - Daily claim streaks and coins

---

## 🎯 Complete Command Count

**Original Commands:** 32
**New Commands:** +11
**Total Commands:** 43

### Breakdown:
- **Fun/Games:** 27 commands
- **Utility:** 16 commands

---

## 🚀 Usage Instructions

1. **Deploy new commands:**
   ```bash
   npm run deploy
   ```

2. **Restart the bot:**
   ```bash
   npm start
   ```

3. **All new commands are ready to use!**

---

## 💡 Key Features

✅ **RPG Elements** - Quests, battles, loot, boss raids
✅ **Economy System** - Coins, daily rewards, streaks
✅ **Social Features** - Quotes, summons, hotseat
✅ **Competitive Gameplay** - Multiple leaderboards
✅ **Persistent Data** - All progress saved
✅ **GDPR Compliant** - Full data deletion support
✅ **Server-Wide Events** - Boss raids
✅ **Engagement Boosters** - Daily streaks, random bonuses

---

**All commands are fully functional and ready to use!** 🎉
