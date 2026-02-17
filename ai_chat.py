"""
Simple AI Chat Service for Discorbo - FREE, No API Keys!
Pattern-based chatbot with personality
"""
import os
import re
import random
from flask import Flask, request, jsonify
from dotenv import load_dotenv

load_dotenv()

app = Flask(__name__)

# Conversation memory
conversations = {}

# Response patterns - add more as needed!
PATTERNS = {
    # Greetings
    r'\b(hi|hello|hey|sup|wassup|yo)\b': [
        "Hey there! 👋 How can I help you today?",
        "Hello! What's up? 😊",
        "Hey! Need help with something?",
        "Hi! I'm Discorbo, your friendly bot! What can I do for you?"
    ],

    # How are you
    r'\b(how are you|how\'s it going|hows it going|whats up|what\'s up)\b': [
        "I'm doing great! Just vibing in the digital realm 🤖",
        "All good! Ready to help you out! 😄",
        "Living my best bot life! What about you?",
        "Fantastic! Thanks for asking! How can I help?"
    ],

    # Bot capabilities
    r'\b(what can you do|commands|help|what do you do)\b': [
        "I have 40+ slash commands! Use `/help` to see them all. I can do trivia, games, utilities, and more! 🎮",
        "Tons of stuff! Try `/help` for the full list. I've got fun games, useful tools, and I love to chat! 💬",
        "I'm packed with features! Type `/help` to explore. Plus, I can chat with you right here! 😊"
    ],

    # Bored/Entertainment
    r'\b(bored|boring|entertain me|something fun)\b': [
        "Let's fix that! Try:\n🎮 `/maze` - Play a maze game\n🧠 `/trivia` - Test your knowledge\n😂 `/meme` - Get a funny meme\n🎲 `/roll` - Roll some dice",
        "Boredom? Not on my watch! Check out `/trivia`, `/maze`, or `/would-you-rather` for some fun!",
        "I got you! Try `/joke` for a laugh, `/vibecheck` to see your vibe, or `/hotseat` to spice things up! 🔥"
    ],

    # Games
    r'\b(game|games|play)\b': [
        "Let's play! I've got:\n🎮 `/maze` - Navigate through mazes\n⚔️ `/battle` - Fight other users\n🎯 `/trivia` - Quiz time\n🎲 `/rps` - Rock Paper Scissors",
        "Game time! Try `/maze`, `/battle`, `/trivia`, or `/coinflip` to get started! 🎮"
    ],

    # Thanks
    r'\b(thanks|thank you|thx|ty)\b': [
        "You're welcome! 😊",
        "No problem! Happy to help! 💙",
        "Anytime! That's what I'm here for! 🤖",
        "Glad I could help! ✨"
    ],

    # Jokes
    r'\b(joke|funny|make me laugh)\b': [
        "Use `/joke` for a good one! I've got tons of jokes programmed in! 😄",
        "Want a joke? Try the `/joke` command - I promise they're better than my dad jokes! 😂"
    ],

    # Love/like
    r'\b(love you|luv u|like you|you\'re (cool|awesome|great))\b': [
        "Aww, you're awesome too! 💙",
        "Thanks! You're pretty cool yourself! 😎",
        "Love you too! Now let's have some fun with my commands! 🎉"
    ],

    # Goodbye
    r'\b(bye|goodbye|see ya|cya|later)\b': [
        "See you later! 👋",
        "Bye! Come back soon! 😊",
        "Catch you later! Have a great day! ✨"
    ]
}

# Default responses for unknown inputs
DEFAULT_RESPONSES = [
    "Interesting! Tell me more, or try `/help` to see what I can do! 🤔",
    "Hmm, I'm not sure about that. Want to try one of my commands? Use `/help`! 💡",
    "I'm just a simple bot, but I've got lots of cool commands! Try `/help` to explore! 🎮",
    "Not sure what you mean, but I'm here to chat or help with commands! Use `/help` to see what I can do! 😊"
]

def get_response(message):
    """Get response based on message pattern matching"""
    message_lower = message.lower().strip()

    # Check patterns
    for pattern, responses in PATTERNS.items():
        if re.search(pattern, message_lower):
            return random.choice(responses)

    # Default response
    return random.choice(DEFAULT_RESPONSES)

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "type": "simple_ai"})

@app.route('/chat', methods=['POST'])
def chat():
    """Process chat message and return response"""
    try:
        data = request.json
        user_id = data.get('user_id')
        username = data.get('username', 'User')
        message = data.get('message', '')
        is_dm = data.get('is_dm', False)

        if not message:
            return jsonify({"error": "No message provided"}), 400

        # Get response
        response = get_response(message)

        # Store in conversation history (optional, for future context)
        if user_id not in conversations:
            conversations[user_id] = []

        conversations[user_id].append({
            "message": message,
            "response": response,
            "timestamp": __import__('time').time()
        })

        # Keep only last 10 interactions
        if len(conversations[user_id]) > 10:
            conversations[user_id] = conversations[user_id][-10:]

        return jsonify({
            "response": response,
            "user_id": user_id,
            "is_dm": is_dm
        })

    except Exception as e:
        print(f"Error processing chat: {e}")
        return jsonify({
            "error": "Internal server error",
            "fallback": "Oops! Something went wrong. Try using `/help` to see my commands! 😅"
        }), 500

@app.route('/clear', methods=['POST'])
def clear_conversation():
    """Clear conversation history for a user"""
    try:
        data = request.json
        user_id = data.get('user_id')

        if user_id in conversations:
            del conversations[user_id]
            return jsonify({"status": "cleared"})

        return jsonify({"status": "no_history"})

    except Exception as e:
        print(f"Error clearing conversation: {e}")
        return jsonify({"error": str(e)}), 500

if __name__ == '__main__':
    port = int(os.environ.get('AI_CHAT_PORT', 5000))
    print("=" * 50)
    print("🤖 Discorbo Simple AI Chat Service")
    print("=" * 50)
    print(f"✅ Running on port {port}")
    print(f"✅ FREE - No API keys needed!")
    print(f"✅ Pattern-based responses")
    print("=" * 50)
    app.run(host='0.0.0.0', port=port, debug=False)
