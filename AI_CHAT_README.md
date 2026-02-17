# FREE AI Chat - No API Keys Needed! 🎉

Simple pattern-based AI that responds to DMs and mentions. **100% FREE, runs locally, zero cost!**

## Quick Start

```bash
# Install Python dependencies (one time)
pip install flask python-dotenv

# Start everything with one command!
npm start
```

That's it! Now mention the bot or DM it.

## Features ✨

- **FREE** - No API keys, no costs!
- **Pattern Matching** - Smart responses to common phrases
- **Expandable** - Easy to add new responses in `ai_chat.py`
- **Memory** - Remembers last 10 interactions per user
- **Personality** - Friendly, helpful, Discord-focused

## Example Conversations

```
You: hey
Bot: Hey there! 👋 How can I help you today?

You: I'm bored
Bot: Let's fix that! Try:
     🎮 /maze - Play a maze game
     🧠 /trivia - Test your knowledge
     😂 /meme - Get a funny meme

You: what can you do
Bot: I have 40+ slash commands! Use /help to see them all. I can do trivia, games, utilities, and more! 🎮

You: thanks!
Bot: You're welcome! 😊
```

## Add Your Own Responses

Edit `ai_chat.py` and add to the `PATTERNS` dictionary:

```python
# Your custom pattern
r'\b(your|trigger|words)\b': [
    "Response option 1",
    "Response option 2",
    "Response option 3"
],
```

The bot will randomly pick from the responses!

## Commands

```bash
npm start          # Start bot + AI (everything!)
npm run start:bot  # Just the Discord bot
npm run start:ai   # Just the AI service
```

## No Cloud, No Costs

- Runs 100% on your machine
- No external APIs
- No rate limits
- No billing surprises

Just pure, simple pattern matching! 🚀
